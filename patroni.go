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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
  "encoding/json"
  "github.com/MakeNowJust/heredoc"
  "github.com/inancgumus/screen"
  "regexp"
  "errors"
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


func Replication_info(user_gucs string) {
 
  fmt.Println("+ Replication information")

  connectionInstance, err := pgx.Connect(context.Background(), ReadConfig())

  if err != nil {

    err_count := 0
    fmt.Println("+ Failover or switchover in progress ?")
    fmt.Println("+ Trying to reconnect every half-second for 20s max")

    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
      //Printout the most legit error message out to screen
      pgerr(err, pgErr,false)
    } else {
       fmt.Print(string(colorRed))
       fmt.Printf("\r+ Try reconnecting to PostgreSQL         : downtime %ds                   ", actual_downtime)
       fmt.Print(string(colorReset))
    }

    for err != nil {
      //we can't connect to the master to grab the information
      //because probably there's a failover ongoing... 
      time.Sleep(500 * time.Millisecond)
      connectionInstance, err = pgx.Connect(context.Background(), ReadConfig())
      err_count = err_count + 1
      if err_count > 40 {
        exit1("\nToo many reconnect failures. Please carrefully check the following error:\n",err)
      }

      if errors.As(err, &pgErr) {
        //Printout the most legit error message out to screen
        pgerr(err, pgErr,false)
      } else {
        fmt.Print(string(colorRed))
        fmt.Printf("\r+ Try reconnecting to PostgreSQL         : downtime %ds                   ", actual_downtime)
        fmt.Print(string(colorReset))
      }
    }
    fmt.Println()
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

  rows, _ := connectionInstance.Query(context.Background(), query)

  defer rows.Close()

  //DEBUG 
  //fmt.Println("replication info query is :",replication_info_query)

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

  if row_count == 0 {
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

  fmt.Println("+----------------------------------+---------------------------------------+----------------------------------+")

  rows.Close()

  err = connectionInstance.Close(context.Background())

  if err != nil {
    exit1("Unable to close connection:\n",err)
	}

}

func patronictloutColorize(input string) string {

  m := regexp.MustCompile("Leader")
  n := regexp.MustCompile("Replica")
  o := regexp.MustCompile("Sync Standby")
  p := regexp.MustCompile("Standby Leader")

  leader     := "${1}"+string(colorRed)   +"Leader"        + string(colorReset)+"$2"
  replica    := "${1}"+string(colorCyan)  +"Replica"       + string(colorReset)+"$2"
  sync_stdby := "${1}"+string(colorGreen) +"Sync Standby"  + string(colorReset)+"$2"
  stdby_lead := "${1}"+string(colorGreen) +"Standby Leader"  + string(colorReset)+"$2"
 
  output := m.ReplaceAllString(input, leader)
  output  = n.ReplaceAllString(output, replica)
  output  = o.ReplaceAllString(output, sync_stdby)
  output  = p.ReplaceAllString(output, stdby_lead)

  return output
}

func PatroniWatch() {

  flag.Parse()
 
  patroni_config := ReadPatroniConfig ()

  if !( patroni_config.Format == "list" || patroni_config.Format ==  "topology") {
    message := "Error : value of Format in "+patroniconfigfilename.value+" must be either 'list' or 'topology' and it's actually set to '"+patroni_config.Format+"'"
    message = message + "\nPlease set one or the other then run again"
    exit1(message,nil)
  }

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
  }

  if patroni_config.Use_sudo == "yes" {
    remote_command = "sudo patronictl -c /etc/patroni/" + patroni_config.Cluster + ".yml " + patroni_config.Format+ " " + patroni_config.Cluster
  } else {
    remote_command = "patronictl -c /etc/patroni/" + patroni_config.Cluster + ".yml " + patroni_config.Format+ " " + patroni_config.Cluster
  }

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

  // Create SSH manager
  sshManager := NewSSHManager(sshConfig)
  defer func() {
    if sshManager.Client != nil {
      sshManager.Client.Close()
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

        if patroni_config.K8s_selector == "" {

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
			      //fmt.Printf("Command '%s' output:\n%s\n", remote_cmd, output)
            patronictlout = string(output)
		      }


        } else {
          //execution on Kubernetes env

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
        }

        err_stop_sec = time.Now().Unix() - err_start_sec

        if patroni_config.Watch_timer > 1 {

          patroni_watch_timer = patroni_config.Watch_timer

          // Clears the screen
          screen.Clear()
          screen.MoveTopLeft()
          fmt.Println()
          currentTime := time.Now()
    
          if patroni_config.Remote_host != "" { 
            fmt.Println("+ Patronictl output from host ", patroni_config.Remote_host, "at", currentTime.Format("2006.01.02 15:04:05"))
          } else {
            fmt.Println("+ Patronictl output from ", pod, "at", currentTime.Format("2006.01.02 15:04:05"))
          }

          //prints out the result of the patronictl list command 
          fmt.Println(patronictloutColorize(patronictlout))

          if patroni_config.Replication_info != "" {
            if !configfilename.set {
              message := "Replication_info set to '"+patroni_config.Replication_info+ "' in " + patroniconfigfilename.value
              message = message + " but no -config <config.json> provided ! Please read documentation."
              exit1(message,nil)
            } else {
              Replication_info(patroni_config.Replication_info)
            }
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

          if patroni_config.Remote_host != "" { 
            fmt.Println("+ Patronictl output from host ", patroni_config.Remote_host, "at", currentTime.Format("2006.01.02 15:04:05"))
          } else {
            fmt.Println("+ Patronictl output from ", pod, "at", currentTime.Format("2006.01.02 15:04:05"))
          }
 
          //prints out the result of the patronictl list command 
          fmt.Println(patronictloutColorize(patronictlout))

          if patroni_config.Replication_info != "" { 
            if !configfilename.set {
              message := "Replication_info set to '"+patroni_config.Replication_info+"' in " + patroniconfigfilename.value
              message = message + " but no -config <config.json> provided ! Please read documentation."
              exit1(message,nil)
            } else {
              Replication_info(patroni_config.Replication_info)
            }
          }
          exit1("Watch_timer in "+patroniconfigfilename.value+" is not >1 so we ran only once",nil)

      } //else: Watch_timer is something inferior to 1 : we run once on
    } // if patroni_config.Watch_timer > 1
  }  //if patroni_config.K8s_selector == "" 
} // func PatroniWatch()
