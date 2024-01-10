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

  //"github.com/jackc/pgx/v5"

const (
	pgReconnectTimeout = 30 * time.Second
)

var (
  //Patroni watcher mode
  remote_command            string
  patroni_watch_timer       int = 0
  patroniconfigfilename     stringFlag
  patronictlout             string = ""
  pod                       string = ""

  replication_info_query = heredoc.Doc(`
select                            
    3
    ,'GUC'
    ,rpad(name,30)
    ,rpad(current_setting(name),70)
    from
      pg_settings where name in (XXX)
  UNION
  select 
    2
    ,rpad((application_name||' Replica (TL:'||(SELECT timeline_id FROM pg_control_checkpoint())||')'),30)
    ,rpad('Sync state : '||sync_state,35)
    ,rpad(coalesce('Write lag  : '||write_lag,'No write lag'),30)
  from  
    pg_stat_replication
  UNION 
  select 
    1
,rpad(regexp_replace(pg_read_file('/etc/hostname'), '\r|\n', '', 'g')::text||(case when pg_is_in_recovery() then ' Replica ' else ' Leader  ' end)||'(TL:'||(SELECT timeline_id FROM pg_control_checkpoint())||')',30)
    ,rpad('Started : '||(to_char(pg_postmaster_start_time(),'YYYY-MM-DD HH24:MI:SS (TZ)')),35)
    ,rpad('Uptime : '||age(current_timestamp,pg_postmaster_start_time()),30)
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
    message := "Error while reading the file provided -patroni "
    message = message + patroniconfigfilename.value +":\n"
    exit1(message,err)
  }

  return configuration
}


func Replication_info(user_gucs string, pgManager *PGManager) {

  fmt.Println("+ Replication information")

  //test connection
	err := pgManager.conn.Ping(context.Background())
	if err != nil {
    fmt.Print(string(colorRed))
    fmt.Println("+ Failover or switchover in progress ?")
    fmt.Println("+ Trying to reconnect every half-second for 20s max")
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

  fmt.Println("+----------------------------------+---------------------------------------+----------------------------------+")

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
      fmt.Println("| ",column1," | ",column2," | ",column3," |")
    } else {
      fmt.Println("| ",column2," | ",column3," |")
    }
  }

/******** JPAREM FIXME 
********* still doesnt work
********* disabling this test while I try to understand 
********* what's going on so I can actually fix it :-)

  if row_count == 0 {
     //row_count can be 0 if the postmaster died while we were running the
     //query, and was alive just before, so the test/ping test was OK.
     //
     //so thats why we 1st try to ping again, and if it pings thats
     //probably because user is not using a superuser connexion, and we will
     //throw the error. 
     // 
     //if the postmaster doesnt ping, we will fall into it in the next
     //iteration
     //
    
     //sleep 1s to avoid pinging a dying server that may ping back
     //while dying..
     //JPAREM doesnt work..
     time.Sleep(1 * time.Second)   

     //test AGAIN connection
	   err := pgManager.conn.Ping(context.Background())
	   if err != nil {
       fmt.Println("+ Failover or switchover in progress ?")
       fmt.Println("+ Trying to reconnect every half-second for 20s max")
		   err := pgManager.PGReconnectWithTimeout(pgReconnectTimeout,err)
		   if err != nil {
         exit1("Failed to reconnect:\n", err)
	     }
     } else {
       //postmaster is alive.. so that's likely the user not using a Superuser
       //connexion
       message :=          "Appears the query returning Replication information from pg_stat_replication\n"
       message = message + "is not returning the data expected. You may have not set a proper Superuser\n"
       message = message + "(e.g. \"postgres\") connection settings in file : "
       message = message + configfilename.value + "\n\n"
       message = message + "If you think your config file is OK, then it's worth filing a bug on\n"
       message = message + "https://github.com/CrunchyData/pgSimload/\n\n"
       message = message + "Meanwhile, to avoid this error you can set Replication_info to \"\" (empty string)\n"
       message = message + "in your Patroni config file : "
       message = message + patroniconfigfilename.value
       exit1(message,nil)
     } 
  }
************/

  fmt.Println("+----------------------------------+---------------------------------------+----------------------------------+")

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

        err_start_sec = time.Now().Unix()

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

        err_stop_sec = time.Now().Unix() - err_start_sec

        if patroni_config.Watch_timer > 1 {

          patroni_watch_timer = patroni_config.Watch_timer

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
  
          //we will sleep around patroni_watch_timer value : it's the goal
          //but the ssh execution time has to be taken in consideration
          //since it can take several seconds to execute
          //so we redefine the watch_timer on the fly to match the goal
          //adapting the program to the SSH execution time
          patroni_watch_timer = patroni_watch_timer - int(err_stop_sec)
	        time.Sleep(time.Duration(patroni_watch_timer) * 1000 * time.Millisecond)

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

        err_start_sec = time.Now().Unix()

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

        err_stop_sec = time.Now().Unix() - err_start_sec

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
  
          //we will sleep around patroni_watch_timer value : it's the goal
          //but the ssh execution time has to be taken in consideration
          //since it can take several seconds to execute
          //so we redefine the watch_timer on the fly to match the goal
          //adapting the program to the SSH execution time
          patroni_watch_timer = patroni_watch_timer - int(err_stop_sec)
	        time.Sleep(time.Duration(patroni_watch_timer) * 1000 * time.Millisecond)

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

  //check if presence of required binaries on the host are present
  // ssh if not running in k8s
  // kubectl if running in k8s
  if patroni_config.K8s_selector == "" {
    //running remotely : ssh has to be present on this system
    if _, err := os.Stat("/usr/bin/ssh"); os.IsNotExist(err) {
      message := "ssh is not present on this system. Please install it prior running"
      message = message + "\npgSimload in Patroni-loop mode against a remote host\n"
      exit1(message,err)
    }

  } else {
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
