package main

import (
	"context"
	"fmt"
  "flag"
	"os"
  "os/exec"
  "time"
  "strings"
	"github.com/eiannone/keyboard"
  "encoding/json"
  "github.com/MakeNowJust/heredoc"
  "github.com/inancgumus/screen"
  "regexp"
)

var (
  //Patroni watcher mode
  patroni_watch_timer       int
  remote_command            string
  patroniconfigfilename     stringFlag
  patronictlout             string = ""
  pod                       string = ""
  // variables to handle exec time 
  // and adjust the loop to match
  // user's expectations
  start                     time.Time

  replication_info_query = heredoc.Doc(`
select                            
    3
    ,'GUC'
    ,rpad(name,32)
    ,rpad(current_setting(name),72)
    from
      pg_settings where name in (XXX)
  UNION
  select 
    2
    ,rpad((application_name||' Replica (TL:'||(SELECT timeline_id FROM pg_control_checkpoint())||')'),32)
    ,rpad('Sync state : '||sync_state,37)
    ,rpad(coalesce('Write lag  : '||write_lag,'No write lag'),32)
  from  
    pg_stat_replication
  UNION 
  select 
    1
,rpad(regexp_replace(pg_read_file('/etc/hostname'), '\r|\n', '',
'g')::text||(case when pg_is_in_recovery() then ' Replica ' else ' Leader  '
end)||'(TL:'||(SELECT timeline_id FROM pg_control_checkpoint())||')',32)
    ,rpad('Started : '||(to_char(pg_postmaster_start_time(),'YYYY-MM-DD HH24:MI:SS (TZ)')),37)
    ,rpad('Uptime : '||age(current_timestamp,pg_postmaster_start_time()),32)
  order by 1,2,3;
  `)
)

type PatroniConfig struct {
    Cluster          string 
    Remote_host      string
    Remote_user      string
    Remote_port      int
    Use_sudo         string
    Ssh_private_key  string
    Replication_info string
    Watch_timer      int
    Format           string
    K8s_selector     string
}


//function to emphase special keywords in the patronictl output
//coloring them
// Leader          will be red
// Replica         will be cyan
// Sync Standby    will be green
// Standby Leader  will be green too
func patronictloutColorize(input string) string {

  m := regexp.MustCompile("Leader")
  n := regexp.MustCompile("Replica")
  o := regexp.MustCompile("Sync Standby")
  p := regexp.MustCompile("Standby Leader")

  leader     := "${1}"+string(colorRed)   +"Leader"         + string(colorReset)+"$2"
  replica    := "${1}"+string(colorCyan)  +"Replica"        + string(colorReset)+"$2"
  sync_stdby := "${1}"+string(colorGreen) +"Sync Standby"   + string(colorReset)+"$2"
  stdby_lead := "${1}"+string(colorGreen) +"Standby Leader" + string(colorReset)+"$2"
 
  output := m.ReplaceAllString(input, leader)
  output  = n.ReplaceAllString(output, replica)
  output  = o.ReplaceAllString(output, sync_stdby)
  output  = p.ReplaceAllString(output, stdby_lead)

  return output
}

// this function is used to sleep for an amount of time that is
// computed between the start of the process to show patronictlout
// with or without Replication info given:
//
// real_exec is the Time the whole thing took
// goal      is an integer(seconds) that is the goal to reach 
//           given by patroni_config.Watch_timer by the user
// 
// once computed, this functions does a sleep() for the right
// amount of time that is computed on each cycle
//
// since the conversions int64 <> Duration are complicated 
// I've put this out of the code in another func() for more
// readability 
func ComputedSleep (real_exec_time time.Duration, goal int) {

  //covert the duration into milliseconds (int64)
  real_exec_time_ms := real_exec_time.Milliseconds()

  //convert the goal(int) into milliseconds (int64)
  watch_timer_ms    := int64(goal) * int64(time.Second) / int64(time.Millisecond)

  //compute the sleep time in ms (int64)
  sleep_time_ms     := watch_timer_ms - real_exec_time_ms

  //if the sleep time is negative, don't sleep, but send a message to the user
  //so he adapts his way too much demanding parameter in patroni_config.Watch_timer
  //because the system can't do that this often..
  if sleep_time_ms < 0 {
    fmt.Print(string(colorRed))
    fmt.Printf("Your system is too slow to do update output each %ds! ",goal)
    fmt.Println("\nPlease raise Watch_timer value in your patroni.json file...")
    fmt.Print(string(colorReset))
  } else {
    //sleep_time_ms is positive so we can sleep a bit then, to match the goal

    //compute the sleep time into Duration in Milliseconds
    sleep_time        := time.Duration(sleep_time_ms) * time.Millisecond

    //DEBUG
    //fmt.Printf("\nDEBUG: Watch_timer              : %dms", watch_timer_ms)
    //fmt.Printf("\nDEBUG: Duration (value)         : %dms", real_exec_time_ms)
    //fmt.Printf("\nDEBUG: Sleep for                : %dms", sleep_time_ms)

    //do sleep a bit now we've computed how long to sleep, to match the goal
	  time.Sleep(sleep_time)
  }
}

// function ReadPatroniConfig() to
// read patroni.json to get parameters about Patroni monitoring mode
// this function gives values to global variables
func ReadPatroniConfig () PatroniConfig {
  flag.Parse()
  file, _ := os.Open(patroniconfigfilename.value)
  defer file.Close()
  decoder := json.NewDecoder(file)
  configuration := PatroniConfig{}
  err := decoder.Decode(&configuration) 
  if err != nil {
    message := "Error while reading the file "
    message = message + patroniconfigfilename.value +":\n"
    exit1(message,err)
  }

  return configuration
}


func Replication_info(user_gucs string, pgManager *PGManager) {

  //test connection
	err := pgManager.conn.Ping(context.Background())
	if err != nil {
    fmt.Print(string(colorRed))
    //fmt.Println("+ Failover or switchover in progress ?")
    //fmt.Println("+ Trying to reconnect every half-second for 20s max")
    fmt.Print(string(colorReset))
		err := pgManager.PGReconnectWithTimeout(pgReconnectTimeout,err)
		if err != nil {
      exit1("Failed to reconnect:\n", err)
	  }
  }

  flag.Parse()

  //replication_info_query is a declared constant, containing XXX
  //string to be replaced by actual content of user setting in the
  //patroni.json file
 
  //if the special value of Replication_info is "nogucs" that indicates
  //the user want the query to be shown, because Replication_info is then
  //NON empty... But doesn't want to show any added GUCS info
  //so by default, gucs gets initialized to ''

  gucs := "''"

  if user_gucs != "nogucs" {
    //user requested the Replication_info extra query to be shown
    //and wants one or more GUCS s.he defined in Replication_info 
    //to be shown in the query output
 
    //since the content of user_gucs (Replication_info actual value passed to
    //that function) is a list of gucs like
    // "server_version, synchronous_commit" (etc..)
    // we have to convert it to 
    // "'server_version', 'synchronous_comit'" (etc)
    // likely adding a simple quote (') before and after the name of the 
    // GUC for the SQL query sent to PostgreSQL to be valid
 
    // add simple quotes before and after each GUC the user want to be
    // show in the Replication information 
    m := regexp.MustCompile(`(\w+)`)
    add_simple_quotes_replace_pattern := "'$0'"
    gucs = m.ReplaceAllString(user_gucs, add_simple_quotes_replace_pattern)

  }

  // replace the "XXX" in the replication_info_query string to put there
  // the GUCS the user want to be shown... or '' if it's "nogucs" so 
  // nothing is sent back (default values of '' to gucs applies then..)
  n := regexp.MustCompile("XXX")
  gucs = "${1}"+ gucs + "$2"
  query := n.ReplaceAllString(replication_info_query, gucs)

  rows, _ := pgManager.conn.Query(context.Background(), query)

  defer rows.Close()

  //DEBUG 
  //fmt.Println("DEBUG : Replication info query is :",replication_info_query)

  output := "+ Replication information\n"
  output = output + "+----------------------------------+---------------------------------------+----------------------------------+\n"

  row_count := 0
 
  for rows.Next() {
    var column0 int
    var column1 string
    var column2 string
    var column3 string

    err := rows.Scan(&column0,&column1, &column2, &column3) 

    if err != nil {
      exit1("Error retrieving replication info:\n",err)
    }

    row_count++

    if column1 != "GUC" {
      output = output + "| " + column1 + " | " + column2 + " | " + column3 + " |\n"
    } else {
      output = output + "| " + column2 + " | " + column3 + " |\n"
    }
  }

  if row_count > 0 {
    output = output + "+----------------------------------+---------------------------------------+----------------------------------+\n"
    fmt.Println(output)
  } else { 
      fmt.Print(string(colorRed))
      fmt.Printf("\nPostgreSQL not responding... Probable failover in progress ?...\n")
      fmt.Print(string(colorReset))
  }

  rows.Close()

}

func PatroniWatch_ssh(patroni_config PatroniConfig, remote_command string, pgManager *PGManager) {

	stopCh := make(chan bool)
	go func() {
		for {
			_, key, err := keyboard.GetKey()
      if err != nil {
        //exit1("Error:\n",err)
			}
			if key == keyboard.KeyEsc {
				stopCh <- true
			}
     }
		}()

  sshConfig := SSHClientConfig{
    Host:       patroni_config.Remote_host,
    Port:       patroni_config.Remote_port,
    User:       patroni_config.Remote_user,
    PrivateKey: patroni_config.Ssh_private_key,
  }

  // Create SSH manager instance
  sshManager := NewSSHManager(sshConfig)
  defer func() {
    if sshManager.Client != nil {
      sshManager.Client.Close()
    }
  }()

  //DEBUG 
  //fmt.Println("DEBUG : SSH Manager instance created")

  loop:
	  for {
		  select {
		  case stop := <-stopCh:
			  if stop {
				  break loop
			  }
		  default:

        start = time.Now()

        //execution on SSH boxes

        output, err := sshManager.RunCommand(remote_command)
    		if err != nil {
           fmt.Print(string(colorRed))
           fmt.Printf("Error executing command remote command \"ssh %v@%v:%v\" :\n  %v", sshConfig.User, sshConfig.Host, sshConfig.Port, remote_command)
           fmt.Print(string(colorReset))
	
	         // If the SSH client is closed, reopen it
			     if sshManager.Client == nil {
             fmt.Print(string(colorRed))
             fmt.Printf("\nTrying to reopen an SSH client...\n")
             fmt.Print(string(colorReset))

				     if err := sshManager.EnsureConnected(); err != nil {
               message := "Error reopening SSH client:\n"
               message = message + "Maybe worth verifying your "+patroniconfigfilename.value + " file ?"
               exit1(message, err)
				     }
			     }
		    } else {
          //DEBUG
			    //fmt.Printf("DEBUG: Command '%s' output:\n%s\n", remote_cmd, output)
          patronictlout = string(output)
		    }


        if patroni_config.Watch_timer > 1 {

          // Clears the screen
          screen.Clear()
          screen.MoveTopLeft()
          fmt.Println()
          currentTime := time.Now()
    
          fmt.Println("+ Patronictl output from host ", patroni_config.Remote_host, "at", currentTime.Format("2006.01.02 15:04:05"))

          //prints out the result of the patronictl list command 
          fmt.Println(patronictloutColorize(patronictlout))

          if pgManager != nil {
            Replication_info(patroni_config.Replication_info, pgManager)
          }
  
          real_exec_time    := time.Since(start)

          //sleep for a computed time to match patroni_config.Watch_timer
          //see ComputedSleep comment for more explanations
          ComputedSleep (real_exec_time, patroni_config.Watch_timer)
           

        } else {

          // Watch_timer is something inferior to 1 : we run once only
  
          // Clears the screen
          screen.Clear()
          screen.MoveTopLeft()
          fmt.Println()
          currentTime := time.Now()

          fmt.Println("+ Patronictl output from host ", patroni_config.Remote_host, "at", currentTime.Format("2006.01.02 15:04:05"))
 
          //prints out the result of the patronictl list command 
          fmt.Println(patronictloutColorize(patronictlout))

          if pgManager != nil { 
              Replication_info(patroni_config.Replication_info, pgManager)
          }
          exit1("Watch_timer in "+patroniconfigfilename.value+" is not >1 so we ran only once",nil)
        
        } //else: Watch_timer is something inferior to 1 : we run once on
      } //select
    } // for
}

func PatroniWatch_k8s(patroni_config PatroniConfig, remote_command string, pgManager *PGManager) {

	stopCh := make(chan bool)
	go func() {
		for {
			_, key, err := keyboard.GetKey()
      if err != nil {
        //exit1("Error:\n",err)
			}
			if key == keyboard.KeyEsc {
				stopCh <- true
			}
     }
		}()


  loop:
	  for {
		  select {
		  case stop := <-stopCh:
			if stop {
				break loop
			}
		  default:

        start = time.Now()

        //initiate error counter  
        err_count := 0

        // get primary pod's name
        command_args := "/usr/bin/kubectl get pods --selector='" + patroni_config.K8s_selector + "' -o name"
        cmd  := exec.Command("sh", "-c", command_args)
        out, err := cmd.CombinedOutput()
        if err != nil {
          //we can't connect to the master pod to grab information 
          //because probably there's a failover/swichover ongoing...
          fmt.Println("+ Failover or switchover in progress ?")
          fmt.Println("+ Waiting for Master pod to be up every second for 20s max")
          for err != nil  {
            time.Sleep(1000 * time.Millisecond)
            cmd := exec.Command("sh", "-c", command_args)
            out, err := cmd.CombinedOutput()
            fmt.Printf(".")
            err_count = err_count + 1
            if err_count > 20 {
              message := "Too many failures. Please carrefully check the following:\n"
              message = message + "Error executing this command:\n"
              message = message + "sh -c \"" + command_args + "\"\n" + string(out) + "\n"
              exit1(message,err)
            }
          }
        }

        //reinitiate error counter in case we face new errors
        err_count = 0

        //no (more) error, we can continue
        pod = strings.ReplaceAll(strings.TrimSpace(string(out)), "\n", "")  
        
        //get patronictl output from master pod 
        command_args = "/usr/bin/kubectl exec -i -c database " + pod + " -- /bin/bash -c 'patronictl -c /etc/patroni/ " + patroni_config.Format + "'"
        cmd = exec.Command("sh","-c",command_args)
        out, err = cmd.CombinedOutput()

	      if err != nil {
          //the patroni isn't answering *yet* on the (new) Primary pod
	        //so we iterate until it does, or exit after 20 retries
          fmt.Println("+ Waiting for patronictl answer from Master every second for 20s max")
          for err != nil  {
            time.Sleep(1000 * time.Millisecond)
            cmd = exec.Command("sh","-c",command_args)
            out, err = cmd.CombinedOutput()
            fmt.Printf(".")
            err_count = err_count + 1
            if err_count > 20 {
              message := "Too many reconnect failures. Please carrefully check the following:\n"
              message = message + "Error executing this command:\n"
              message = message + "sh -c \"" + command_args + "\"\n" + string(out) + "\n"
              exit1(message,err)
            }
          }
	      }

        patronictlout = string(out)

        if patroni_config.Watch_timer > 1 {

          patroni_watch_timer = patroni_config.Watch_timer

          // Clears the screen
          screen.Clear()
          screen.MoveTopLeft()
          fmt.Println()
          currentTime := time.Now()
    
          fmt.Println("+ Patronictl output from ", pod, "at", currentTime.Format("2006.01.02 15:04:05"))

          //prints out the result of the patronictl list command 
          fmt.Println(patronictloutColorize(patronictlout))

          if pgManager != nil {
              Replication_info(patroni_config.Replication_info, pgManager)
          }
  
          real_exec_time :=  time.Since(start)

          //sleep for a computed time to match patroni_config.Watch_timer
          //see ComputedSleep comment for more explanations
          ComputedSleep (real_exec_time, patroni_config.Watch_timer)

        } else {

          // Watch_timer is something inferior to 1 : we run once only
  
          // Clears the screen
          screen.Clear()
          screen.MoveTopLeft()
          fmt.Println()
          currentTime := time.Now()

          fmt.Println("+ Patronictl output from ", pod, "at", currentTime.Format("2006.01.02 15:04:05"))
 
          //prints out the result of the patronictl list command 
          fmt.Println(patronictloutColorize(patronictlout))

          if pgManager != nil {
              Replication_info(patroni_config.Replication_info, pgManager)
          }

          exit1("Watch_timer in "+patroniconfigfilename.value+" is not >1 so we ran only once",nil)

        } // if patroni_config.Watch_timer > 1
    } //select
  } //for
} // func PatroniWatch_k8s()

func PatroniWatch() {

  flag.Parse()
 
  patroni_config := ReadPatroniConfig ()

  if !( patroni_config.Format == "list" || patroni_config.Format ==  "topology") {
    message := "Error : value of Format in "+patroniconfigfilename.value+" must be either 'list' or 'topology' and it's actually set to '"+patroni_config.Format+"'"
    message = message + "\nPlease set one or the other then run again"
    exit1(message,nil)
  }

  if patroni_config.Use_sudo == "yes" {
    remote_command = "sudo patronictl -c /etc/patroni/" + patroni_config.Cluster + ".yml " + patroni_config.Format+ " " + patroni_config.Cluster
  } else {
    remote_command = "patronictl -c /etc/patroni/" + patroni_config.Cluster + ".yml " + patroni_config.Format+ " " + patroni_config.Cluster
  }

  //default mode
  mode := "ssh"

  //check if presence of required binary kubectl on the host
  //ssh binary not necessary, handled by golang directly
  if patroni_config.K8s_selector != "" {
    //running localy with kubectl
    if _, err := os.Stat("/usr/bin/kubectl"); os.IsNotExist(err) {
      message := "kubectl is not present on this system. Please install it prior running"
      message = message + "\npgSimload in Patroni-loop mode against a k8s env\n"
      exit1(message,err)
    }
    mode = "k8s"
  }
 
  if patroni_config.Replication_info != "" {
    //execute with replication info : PG connex needed
    // Create PG Manager instance
    
    //DEBUG
    //fmt.Println("DEBUG: New PG Manager instance") 

    pgManager, err := NewPGManager(configfilename.value)
    if err != nil {
      exit1("Failed to create PGManager:\n", err)
    }

    // Initial connection
    conn, err := pgManager.PGConnect()
    if err != nil {
      //we won't try to reconnect here, since GUCS generation file
      //is an unitary operation...
      exit1("Failed to connect to PostgreSQL:\n", err)
    }
    defer conn.Close(context.Background())

    if mode == "ssh" {
      //mode = "ssh"
      PatroniWatch_ssh(patroni_config, remote_command, pgManager)
    } else {
      //mode = "k8s"
      PatroniWatch_k8s(patroni_config, remote_command, pgManager)
    }
  } else {
    //execute without replication info : no PG connex needed

    if mode == "ssh" {
      //mode = "ssh"
      PatroniWatch_ssh(patroni_config, remote_command, nil)
    } else {
      //mode = "k8s"
      PatroniWatch_k8s(patroni_config, remote_command, nil)
    }

  } 

} // func PatroniWatch()
