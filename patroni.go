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
  "regexp"
)

var (
  //Patroni watcher mode
  patroni_watch_timer       int
  remote_command            string
  patroniconfigfilename     stringFlag
  patronictlout             string = ""
  pod                       string = ""

  rep_info_replicas = heredoc.Doc(`
  select
      rpad(application_name,32) as pod
    , rpad(sync_state,18) as role
    , (SELECT rpad(timeline_id::text,5) FROM pg_control_checkpoint()) as tl
    , rpad(coalesce('Write lag  : '||write_lag,'No write lag'),32) as lag
  from
    pg_stat_replication
  order by 1,2;
  `)


  rep_info_gucs = heredoc.Doc(`
    select
        rpad(name,32)                 as name
      , current_setting(name)         as setting
    from
      pg_settings where name in (XXX)
    order by 1;
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
    K8s_namespace    string
    K8s_selector     string
}


//function to emphase special keywords in the patronictl output
//coloring them
// Leader          will be red
// Replica         will be cyan
// Sync Standby    will be green
// Standby Leader  will be green too
func patronictloutColorize(input string) string {

  m := regexp.MustCompile("Standby Leader")
  n := regexp.MustCompile("Leader")
  o := regexp.MustCompile("Replica")
  p := regexp.MustCompile("Sync Standby")

  stdby_lead := "${1}"+string(colorRed)   +"Standby Leader" + string(colorReset)+"$2"
  leader     := "${1}"+string(colorRed)   +"Leader"         + string(colorReset)+"$2"
  replica    := "${1}"+string(colorCyan)  +"Replica"        + string(colorReset)+"$2"
  sync_stdby := "${1}"+string(colorGreen) +"Sync Standby"   + string(colorReset)+"$2"
 
  output := m.ReplaceAllString(input,  stdby_lead)
  output  = n.ReplaceAllString(output, leader)
  output  = o.ReplaceAllString(output, replica)
  output  = p.ReplaceAllString(output, sync_stdby)

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
    message += patroniconfigfilename.value +":\n"
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

  //SHOW REPLICA(s) INFO

  rows, _ := pgManager.conn.Query(context.Background(), rep_info_replicas)
   
  defer rows.Close()

  
  output  = "+ Replica(s) information ----------+--------------------+-------+----------------------------------+\n"
  output += "| Member                           | Role               | TL    | Lag                              |\n"
  output += "+----------------------------------+--------------------+-------+----------------------------------+\n"

  row_count := 0
 
  for rows.Next() {
    var column1 string
    var column2 string
    var column3 string
    var column4 string

    err := rows.Scan(&column1, &column2, &column3, &column4)

    if err != nil {
      exit1("Error retrieving Leader info:\n",err)
    }

    row_count++

    output += "| "  + column1 
    output += " | " + column2
    output += " | " + column3 
    output += " | " + column4 + " |\n"
    
  }

  if row_count > 0 {
    output += "+----------------------------------+--------------------+-------+----------------------------------+\n"
    fmt.Println(output)
  } else { 
      fmt.Print(string(colorRed))
      fmt.Printf("\nPostgreSQL not responding... Probable failover in progress ?...\n")
      fmt.Print(string(colorReset))
  }

  rows.Close()

  //SHOW GUCS

  //rep_info_gucs is a declared constant, containing XXX
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

    // replace the "XXX" in the replication_info_query string to put there
    // the GUCS the user want to be shown... or '' if it's "nogucs" so 
    // nothing is sent back (default values of '' to gucs applies then..)
    n := regexp.MustCompile("XXX")
    gucs = "${1}"+ gucs + "$2"
    query := n.ReplaceAllString(rep_info_gucs, gucs)

    rows, _ = pgManager.conn.Query(context.Background(), query)

    defer rows.Close()

    //DEBUG 
    //fmt.Println("DEBUG : Replication info query is :",replication_info_query)

    output = "+ GUCs information\n"

    for rows.Next() {
      var column1 string
      var column2 string

      err := rows.Scan(&column1, &column2)

      if err != nil {
        exit1("Error retrieving GUCs info:\n",err)
      }

      row_count++

      output = output + " + " + column1 + " : " + column2 + "\n"
    
    }

    if row_count > 0 {
      fmt.Println(output)
    } else { 
        fmt.Print(string(colorRed))
        fmt.Printf("\nPostgreSQL not responding... Probable failover in progress ?...\n")
        fmt.Print(string(colorReset))
    }

    rows.Close()

  } // End of    if user_gucs != "nogucs" {

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


        // variables to handle exec time 
        // and adjust the loop to match
        // user's expectations
        start := time.Now()

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
               message += "Maybe worth verifying your "+patroniconfigfilename.value + " file ?"
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
          fmt.Printf("\x1bc")

          //useless new line... fmt.Println()
          currentTime := time.Now()
    
          fmt.Println("+ Patronictl output from host", patroni_config.Remote_host, "at", currentTime.Format("2006.01.02 15:04:05"))
          fmt.Println()

          //prints out the result of the patronictl list command 
          fmt.Println(patronictloutColorize(patronictlout))

          if pgManager != nil {
            Replication_info(patroni_config.Replication_info, pgManager)
          }
 
          //it took actually that real_exec_time to    
          //execute this step in the main loop
          real_exec_time    := time.Since(start)

          //sleep for a computed time to match patroni_config.Watch_timer
          //see ComputedSleep comment for more explanations
          ComputedSleep (real_exec_time, patroni_config.Watch_timer)
           

        } else {

          // Watch_timer is something inferior to 1 : we run once only
  
          // Clears the screen
          fmt.Printf("\x1bc")

          //useless new line at start of screen 
          //fmt.Println()
          currentTime := time.Now()

          fmt.Println("+ Patronictl output from host", patroni_config.Remote_host, "at", currentTime.Format("2006.01.02 15:04:05"))
          fmt.Println()

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

        // variables to handle exec time 
        // and adjust the loop to match
        // user's expectations
        start := time.Now()

        // get primary pod's name
        command_args := "kubectl get pods" 
        command_args += " -n " + patroni_config.K8s_namespace
        command_args += " --selector='" + patroni_config.K8s_selector+"'"
        command_args += " -o name"
        cmd  := exec.Command("sh", "-c", command_args)
        out, err := cmd.CombinedOutput()

        if err != nil {
          //we can't connect to the master pod to grab information 
          //because probably there's a failover/swichover ongoing...
          message := "An error happened while trying to get the Primary's pod name"
          message += "\nError executing this command:\n"
          message += "sh -c \"" + command_args + "\"\n" + string(out) + "\n"
          exit1(message, err)
        }

        //no (more) error, we can continue
        pod = strings.ReplaceAll(strings.TrimSpace(string(out)), "\n", "")  

        //maybe the pod is not up! In such case, pod is an empty string !
        if pod == "" { 
          patronictlout = "+ Primary PostgreSQL pod is not up yet"
        } else {
        
          //get patronictl output from master pod 
          command_args =  "kubectl "
          command_args += " -n " + patroni_config.K8s_namespace
          command_args += " exec -i -c database " + pod
          command_args += " -- /bin/bash -c 'patronictl -c /etc/patroni/ " + patroni_config.Format + "'"
          cmd = exec.Command("sh","-c",command_args)
          out, err = cmd.CombinedOutput()

	        if err != nil {
            //the patroni isn't answering *yet* on the (new) Primary pod
            fmt.Println("+ Waiting for patronictl answer from Primary")
	        }
          patronictlout = string(out)
        }

        if patroni_config.Watch_timer > 1 {

          patroni_watch_timer = patroni_config.Watch_timer

          // Clears the screen
          fmt.Printf("\x1bc")

          currentTime := time.Now()
   
          if pod=="" { 
            fmt.Println("+ Patronictl output at", currentTime.Format("2006.01.02 15:04:05"))
            fmt.Println()
          } else {
            fmt.Println("+ Patronictl output from", pod, "at", currentTime.Format("2006.01.02 15:04:05"))
            fmt.Println()
          }

          //prints out the result of the patronictl list command 
          fmt.Println(patronictloutColorize(patronictlout))

          if pgManager != nil {
              Replication_info(patroni_config.Replication_info, pgManager)
          }
 
          // it took actually that real_exec_time to execute that
          // step in the main loop 
          real_exec_time :=  time.Since(start)

          //sleep for a computed time to match patroni_config.Watch_timer
          //see ComputedSleep comment for more explanations
          ComputedSleep (real_exec_time, patroni_config.Watch_timer)

        } else {

          // Watch_timer is something inferior to 1 : we run once only
  
          // Clears the screen
          fmt.Printf("\x1bc")

          //useless newline at start of screen
          // fmt.Println()
          currentTime := time.Now()

          if pod=="" {
            fmt.Println("+ Patronictl output at", currentTime.Format("2006.01.02 15:04:05"))
            fmt.Println()
          } else {
            fmt.Println("+ Patronictl output from", pod, "at", currentTime.Format("2006.01.02 15:04:05"))
            fmt.Println()
          }
 
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
    message += "\nPlease set one or the other then run again"
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

    // try to run kubectl once to insure it is installed
    cmd := exec.Command("kubectl")
    err := cmd.Run()
    if err != nil {
      message := "kubectl is not present on this system. Please install it prior running"
      message = message + "\npgSimload in Kube-watcher mode against a k8s env\n"
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
