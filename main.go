package main

import (
	"context"
	"fmt"
  "flag"
	"os"
  "os/exec"
  "time"
  "io/ioutil"
  "strings"
	"github.com/eiannone/keyboard"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
  "encoding/json"
  "github.com/MakeNowJust/heredoc"
  "golang.org/x/crypto/ssh"
  "bytes"
  "net"
  "github.com/inancgumus/screen"
  "regexp"
  "errors"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
  colorYellow = "\033[33m"
  colorBlue   = "\033[34m"
  colorPurple = "\033[35m"
  colorCyan   = "\033[36m"
  colorWhite  = "\033[37m"

  ClearLine  = "\033[2K"

  MoveCursorCol1 = "\033[1G"
);

var (
	i                int = 0
  disconnected     bool = false
  previous_loop_err bool = false
	success_count    int64 = 0
	errors_count     int64 = 0
  err_start_sec    int64 = 0
  err_stop_sec     int64 = 0
  total_start_sec  int64 = 0
  actual_downtime  int64 = 0
  total_downtime   int64 = 0
  total_exec_time  int64 = 0
  statements_per_sec int64 =0
  //DEBUG
  //error_log        string = ""

	connectionInstance *pgx.Conn // this is "internal", as in: we should NOT use this directly
  configfilename            stringFlag
  createfilename            stringFlag
  scriptfilename            stringFlag
  //Session Parameters
  sessiongucs               string = ""
  sessiongucsfilename       stringFlag
  gathergucsfilename        stringFlag
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

  gather_gucs_query = heredoc.Doc(`
select 
    E'    {\n'
   ||'      "parameter" : "'||name||E'\",\n'
   ||'      "value"     : "'||current_setting(name)||E'"\n    }'
from pg_settings 
where 
      context = 'user'
  and name not in ('application_name','search_path')
ORDER BY 1;
  `)
 
  //upper : we could grab also "superuser" parameters but..
  //context in ('superuser','user') is only shown
  //FIXME : search_patch causes problems because of the $user thing
  //        we should do some special treatments somewhere to avoid
  //        this but since this info is of poor interest in the
  //        pgSimload usage, we let it aside for now

  gather_gucs_file_header = heredoc.Doc(`
{
  "sessionparameters": [
`)

  gather_gucs_file_footer = heredoc.Doc(`
  ]
}
`)

  Version = "pgSimLoad v.1.0.0 - December 8th 2023"

  License = heredoc.Doc(`
**The PostgreSQL License**

Copyright (c) 2022-2023, Crunchy Data Solutions, Inc.

Permission to use, copy, modify, and distribute this software and its
documentation for any purpose, without fee, and without a written agreement is
hereby granted, provided that the above copyright notice and this paragraph
and the following two paragraphs appear in all copies.

IN NO EVENT SHALL CRUNCHY DATA SOLUTIONS, INC. BE LIABLE TO ANY PARTY FOR
DIRECT, INDIRECT, SPECIAL, INCIDENTAL, OR CONSEQUENTIAL DAMAGES, INCLUDING
LOST PROFITS, ARISING OUT OF THE USE OF THIS SOFTWARE AND ITS DOCUMENTATION,
EVEN IF CRUNCHY DATA SOLUTIONS, INC. HAS BEEN ADVISED OF THE POSSIBILITY OF
SUCH DAMAGE.

CRUNCHY DATA SOLUTIONS, INC. SPECIFICALLY DISCLAIMS ANY WARRANTIES, INCLUDING,
BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE. THE SOFTWARE PROVIDED HEREUNDER IS ON AN "AS IS" BASIS,
AND CRUNCHY DATA SOLUTIONS, INC. HAS NO OBLIGATIONS TO PROVIDE MAINTENANCE,
SUPPORT, UPDATES, ENHANCEMENTS, OR MODIFICATIONS.

For any question reach programmer at jean-paul.argudo@crunchydata.com.`)

  License_short_notice = heredoc.Doc(`Copyright (c) 2022-2023, Crunchy Data Solutions, Inc.
  This program is licensied under The PostgreSQL License. You have a copy
  of the full license aside the source code in the file named LICENSE.md.`)

  Contact = "You can contact programmer at : Jean-Paul Argudo <jean-paul.argudo@crunchydata.com>"
)

type Config struct {
     Hostname        string
	   Port            string
	   Database        string
	   Username        string
	   Password        string
     Sslmode         string
     ApplicationName string
}

type PatroniConfig struct {
    Cluster          string 
    Remote_host      string
    Remote_user      string
    Use_sudo         string
    Ssh_private_key  string
    Replication_info string
    Watch_timer      int
    Format           string
    K8s_selector     string
}

type Queries struct {
    Queries []Query 
}

type Query struct {
     DDL_SQL    string 
     Comment    string 
}

type SessionParameters struct {
    SessionParameters []SessionParameter
}

type SessionParameter struct {
     Parameter  string
     Value      string
}

type stringFlag struct {
    set   bool
    value string
}

func (sf *stringFlag) Set(x string) error {
    sf.value = x
    sf.set = true
    return nil
}

func (sf *stringFlag) String() string {
    return sf.value
}

// Function to exit(1) the program putting in red the error message
func exit1(message string, errcode error) {
    fmt.Print(string(colorRed))
    if errcode== nil {
      fmt.Println(message)
    } else {
      fmt.Println(message,errcode)
    }
    _ = keyboard.Close()
    fmt.Print(string(colorReset))
    os.Exit(1)
}

//function to manage pgErr codes
func pgerr(err error, pgErr *pgconn.PgError, newline bool ) {

  //we have error code starting by 25* : server is in read_only
  //because of switchover of failover, things get aborted
  //we explicitly say that to the user and we're going to wait
  //1sec more for the server to recover
  //
  // Full list of error codes:
  //   https://www.postgresql.org/docs/current/errcodes-appendix.html


  if errors.As(err, &pgErr) {
    var message string
    actual_downtime = time.Now().Unix()-err_start_sec

    switch (pgErr.Code) {
     case "25P01","25P02","25P03","25006":
        //fmt.Printf(  "\r[25...] PG server in recovery mode    : Actual downtime %d seconds             ", actual_downtime)
        message = "PG server in recovery mode    "
     case "28000", "28P01":
        message = "Invalid auth                  "
     case "57P01":
        message = "PG terminated by admin cmd    "
     case "57P02":
        message = "PG crash shutdown             "
     case "57P03":
        message = "Cannot connect now            "
     case "57P04":
        message = "Database dropped              "
     case "57P05":
        message = "Idle session timeout          "
     case "42601":
        message = "Syntax error in SQL           "
     default:
        message = "Other error from PG           " 
     }

      if newline { 
        fmt.Println() 
      }

      fmt.Print(string(colorRed))
      fmt.Printf("\r[%s] %s : downtime %ds                     ", pgErr.Code,message, actual_downtime)
      fmt.Print(string(colorReset))
   }
}


// Function to gather modifiable per-session GUCS, included contexts
// 'user' and 'superuser', because we don't know in advance what will
// be the user defined in the config.json file... Will be up to the
// user to try to modify things in the session_parameter.json file 
// that this function will write into (gathergucsfilename.value) 
func gatherGucs () {

  connectionInstance, err := pgx.Connect(context.Background(), ReadConfig())

  if err != nil {
    exit1("Could not connect to Postgres:\n",err)
  }

  flag.Parse()

  var file string = gather_gucs_file_header

  rows, _ := connectionInstance.Query(context.Background(), gather_gucs_query)

  defer rows.Close()

  for rows.Next() {

    var parameter_json string

    err := rows.Scan(&parameter_json)

    if err != nil {
      exit1("Error retrieving GUCs from PostgreSQL server:\n",err)
    }
 
    if err := rows.Err(); err != nil {
      exit1("Error:\n",err)
    }

    file = file + parameter_json + ",\n"
    i = i+1
  } 

  file = file[:len(file)-2]
  file = file + "\n" + gather_gucs_file_footer

  rows.Close()
 
	err = ioutil.WriteFile(gathergucsfilename.value, []byte(file), 0644)

  err = connectionInstance.Close(context.Background())

  if err != nil {
    exit1("Unable to close connection\n",err)
  }

  fmt.Println("Session parameters template file created !")
  fmt.Println("You can now edit "+gathergucsfilename.value+" to suit your needs")
  fmt.Println("to be used afterwards with -session_parameters in SQL-loop mode")

}


// Function to do a remote run of a command thru an ssh connection
func remoteRun(user string, addr string, privateKey string, cmd string) (string, error) {


    // privateKey could be read from a file, or retrieved from another storage
    // source, such as the Secret Service / GNOME Keyring
    file_privateKey, err := os.ReadFile(privateKey)
	  if err != nil {
      message := "Could not read SSH private key file defined in " 
      message = message + patroniconfigfilename.value +  ":\n"
      exit1(message,err)
	  }
  
    key, err := ssh.ParsePrivateKey([]byte(file_privateKey))
    if err != nil {
      message := "Could not use SSH private key file defined in "
      message = message + patroniconfigfilename.value
      message = message + "\nYou may have entered the public key instead the private one ?\n"
      exit1(message,err)
    }

    // Authentication
    config := &ssh.ClientConfig{
        User: user,
        // https://github.com/golang/go/issues/19767 
        // as clientConfig is non-permissive by default 
        // you can set ssh.InsercureIgnoreHostKey to allow any host 
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),
        Auth: []ssh.AuthMethod{
            ssh.PublicKeys(key),
        },
        //alternatively, you could use a password
        /*
            Auth: []ssh.AuthMethod{
                ssh.Password("PASSWORD"),
            },
        */
    }
    // Connect
    client, err := ssh.Dial("tcp", net.JoinHostPort(addr, "22"), config)
    if err != nil {
        return "", err
    }
    // Create a session. It is one session per command.
    session, err := client.NewSession()
    if err != nil {
        return "", err
    }

    //last-in-first-out order
    //DEFERing the 
    //...close the ssh.Dial but
    defer client.Close()
    //...before, close the client.NewSession
    defer session.Close()

    var b bytes.Buffer  // import "bytes"
    session.Stdout = &b // get output
    // you can also pass what gets input to the stdin, allowing you to pipe
    // content from client to server
    //      session.Stdin = bytes.NewBufferString("My input")

    // Finally, run the command
    err = session.Run(cmd)
    return b.String(), err
 
}

func init() {
    flag.Var(&configfilename,        "config",       "JSON config filename")
    flag.Var(&createfilename,        "create",       "JSON create filename")
    flag.Var(&scriptfilename,        "script",       "SQL script filename")
    flag.Var(&sessiongucsfilename,   "session_parameters", "JSON session gucs filename")
    flag.Var(&patroniconfigfilename, "patroni",      "JSON Patroni watcher mode config filename")
    flag.Var(&gathergucsfilename   , "create_gucs_template", "outputs to that JSON filename")
}

//function to check flags passed with --flag value
//upon execution of the tool
func CheckFlags () {
 
  help    := flag.Bool("help", false, "display some help")
  version := flag.Bool("version", false, "display version")
  license := flag.Bool("license", false, "display license")
  contact := flag.Bool("contact", false, "display where to contact programmers")  

  flag.Parse()

  if *version {
		_ = keyboard.Close()
    fmt.Printf("%s\n",Version);
    os.Exit(0);
  }
   
  if *license {
		_ = keyboard.Close()
    fmt.Printf("%s is licensed under \n",Version);
    fmt.Printf("%s\n",License);
    os.Exit(0);
  }
 
  if *contact {
		_ = keyboard.Close()
    fmt.Printf("%s\n",Contact);
    os.Exit(0);
  }

  if *help {
		_ = keyboard.Close()
    fmt.Println("Please read documentation in doc/README.md");
    os.Exit(0);
  } 

  if gathergucsfilename.set {
    if !configfilename.set {
      message := "To create a template JSON file to be used in -session_parameters\n"
      message = message + "You actually have to use a -config config.json in conjunction with it\n"
      exit1(message,nil)
    } else {
      gatherGucs()
      _ = keyboard.Close()
      os.Exit(0)
    }
  } 

  if !patroniconfigfilename.set {
    
    if (!configfilename.set || !scriptfilename.set) {
      fmt.Print(string(colorRed))
      fmt.Println("You miss one parameter to run pgSimLoad properly in SQL-loop mode:")

      message := "Please read documentation in doc/README.md"
 
      if !configfilename.set {
        exit1("-config is not set !\n"+message,nil)
      }

      if !scriptfilename.set {
        exit1("-script is not set !\n"+message,nil)
      }
    } 
  }

}

// function ReadConfig() to
// read config.json to get database credentials 
// returns an string formated enough to connect to PostgreSQL
func ReadConfig() string {
  flag.Parse()
  file, _ := os.Open(configfilename.value)
  defer file.Close()
  decoder := json.NewDecoder(file)
  configuration := Config{}
  err := decoder.Decode(&configuration)
  if err != nil {
    message := "Error while parsing the JSON file provided in -config "
    message = message + configfilename.value + ":\n"
    exit1(message,err)
  }
  //going with URI like this works better with TLS environments,
  conn_uri := "postgresql://"+configuration.Username+":"+configuration.Password+"@"+configuration.Hostname+":"+configuration.Port+"/"+configuration.Database+"?application_name="+configuration.ApplicationName
  return conn_uri
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

  // We're looping on an ssh remote command where a Patroni of the cluster is
  // running
  // User can break the loop entering the <ESC> key
  // We're looping with a delay of "--watch_timer x " seconds
  // created by a "sleep" command of a computed value
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

        if patroni_config.K8s_selector == "" {
          //execution on bare metal or VMs : ssh the machine and run
          //patronictl there
          output, err := remoteRun(patroni_config.Remote_user, patroni_config.Remote_host, patroni_config.Ssh_private_key, remote_command)

          if err != nil {
            message := "Error while running the remote command :\n  " + remote_command + "\n  executed as " + patroni_config.Remote_user 
            message = message + "\n  on host" + patroni_config.Remote_host + "\n  using SSH private_key " 
            message = message + patroni_config.Ssh_private_key + ":\n  error returned was :\n"
            exit1(message,err)
          }

          patronictlout = string(output)

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
        }
     }
  }
}


func ExecCreate() {

	connectionInstance, err := pgx.Connect(context.Background(), ReadConfig())
	if err != nil {
    exit1("Could not connect to Postgres:\n", err)
	}

  flag.Parse()

  // Open our create.json script with DDLs to run prior
  // to loop on excuting script.sql queries
  create_ddl_file, err := os.Open(createfilename.value)
  
  // if we os.Open returns an error then handle it
  if err != nil {
	  exit1("Could not open DDL script:\n" , err)
  }

  fmt.Println("Executing DDL Script :")

  // defer the closing of our jsonFile so that we can parse it later on
  defer create_ddl_file.Close()

  // read our opened xmlFile as a byte array.
  byteValue, _ := ioutil.ReadAll(create_ddl_file)

  // we initialize our Queries array
  var q Queries

  // we unmarshal our byteArray which contains our
  // jsonFile's content into 'Queries' which we defined above
  json.Unmarshal(byteValue, &q)

  // we iterate through every query within our Query array and
  // print out the query DDL and Comment
  for i := 0; i < len(q.Queries); i++ {
    //DEBUG 
    //fmt.Println("SQL Query : " + q.Queries[i].DDL_SQL)
    //DEBUG 
    //fmt.Println("Comment   : " + q.Queries[i].Comment)
    _, err := connectionInstance.Exec(context.Background(),q.Queries[i].DDL_SQL)

    if err != nil {
      //connectionInstance.Close(context.Background())
      message := "Something went wrong trying to execute the SQL script\n"
      message = message + createfilename.value + " on the database\n"
      exit1(message,err)
    }

    fmt.Printf("   %q\n",q.Queries[i].Comment)
  }

  fmt.Printf("Script %q successfully executed !\n",createfilename.value)
	err = connectionInstance.Close(context.Background())
  if err != nil {
    exit1("Unable to close connection:\n",err)
	}
}


func connectToDB() (*pgx.Conn, error) {
	connectionInstance, err := pgx.Connect(context.Background(), ReadConfig())
	if err != nil {
    //DEBUG
    //fmt.Println("Connection URI was : ",ReadConfig()) 
    exit1("Could not connect to Postgres:\n",err)
	}

  fmt.Printf("Successfully connected to database")
	//return conn, nil
	return connectionInstance, nil
}


func run_simload() {

	connectionInstance, err := pgx.Connect(context.Background(), ReadConfig())
	if err != nil {
		exit1("Could not connect to Postgres:\n",err)
	}

  //A session GUCs file has to be used 
  // whenever sessiongucs.set
  if sessiongucs != "" {
    //DEBUG
    //fmt.Println("Session GUCS",sessiongucs)
    _, err := connectionInstance.Exec(context.Background(), sessiongucs)

    if err != nil { 
      exit1("Error while trying to set session parameters as described in -session_parameters " + sessiongucsfilename.value+"\n",err)
    }
  }

  // read script.sql
  script_file, err := ioutil.ReadFile(scriptfilename.value)

  if err != nil {
    exit1("Could not read script file:\n" , err)
  }

  statements := strings.Split(string(script_file), ";\n")

  // last element is an empty line
  statements = statements[:len(statements)-1]

	fmt.Println()
	//fmt.Println("Statements for main loop loaded!")
  fmt.Printf("Now entering the main loop, executing script %q\n",scriptfilename.value) 

	// MAIN loop on your command(s) in script.sql
	// This is to be able to stop the loop on <Esc>
	stopCh := make(chan bool)
	go func() {
    total_start_sec = time.Now().Unix()
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

			// Check if connection closed
			// Try reconnect every 0.5 second (500 ms) if closed
			// if connection successful execute statement(s)
			// else continue to loop until connection or esc
			if connectionInstance.IsClosed() {

				time.Sleep(500 * time.Millisecond)

        //this re-connection can take time and it's not tracked
        //so the actual_downtime shown on screen may look stuck
        //but at the moment, I don't know how to keep track of this
        //Thus, the real dowtime of the whole downtime event will 
        //be correctly shown on the message "Reconnected after xxx seconds of
        //dontime"
		 		tmpConn, err := pgx.Connect(context.Background(), ReadConfig())

				if err == nil {
          
          //we succeeded in reconnecting
					connectionInstance = tmpConn
          disconnected = false

          actual_downtime = time.Now().Unix()-err_start_sec

          fmt.Printf(ClearLine);
          fmt.Printf(MoveCursorCol1);
          fmt.Print(string(colorGreen))
          fmt.Printf("Reconnected after %ds PG downtime", actual_downtime)
          fmt.Print(string(colorReset))
 

          // A session GUCs file has to be used 
          // whenever sessiongucs.set : we have to apply this if we're here
          // since the session has been reset
          if sessiongucs != "" {
            _, err := connectionInstance.Exec(context.Background(), sessiongucs)
            if err != nil { 
              message := "Error while trying to set session parameters as described in -session_parameters "
              message = message + sessiongucsfilename.value + " :\n" 
              exit1(message,err)
            } else {
              fmt.Print(string(colorGreen))
              fmt.Printf("\nSession parameters aplied to the session")
              fmt.Print(string(colorReset))
            }
				  } 
			  } else {

          //we didn't succeed to reconnect
          disconnected = true

          //if we weren't into trouble until now... now we are 
          //so let's start the timer to keep track when 
          //that happened
          if err_start_sec == 0 {
            err_start_sec = time.Now().Unix()
          }
 
          //update the actual downtime 
          actual_downtime = time.Now().Unix()-err_start_sec

          var pgErr *pgconn.PgError
          if errors.As(err, &pgErr) {
            //DEBUG
            //fmt.Println(pgErr.Message) // => syntax error at end of input
            //fmt.Println(pgErr.Code)    // => returns the error code, eg 42601

            //Printout the most legit error message out to screen
            //DEBUG pgerr(err, pgErr,true)
            pgerr(err, pgErr,false)

            //DEBUG
            //error_log = error_log + "\n[" + time.Now().Format("2006-01-02 15:04:05") + "]" + "[" + pgErr.Code + "] " + pgErr.Message
          } else {
                fmt.Print(string(colorRed))
                fmt.Printf("\rTry reconnecting to PostgreSQL         : downtime %ds                   ", actual_downtime)
                fmt.Print(string(colorReset))
          }
        }
      }

      if !disconnected {
  
        var previous_stmt_error bool = false

        for _, statement := range statements {

          // we first lowercase the statement which facilitates a later search
          // for insert, delete, update and select in a later regexp
          // because we only count those statemets in the counters
          statement = strings.ToLower(statement)

          //we execute each statement, one by one
          _, err := connectionInstance.Exec(context.Background(), statement)

          if err != nil { 
            //the last statement is in error

            //if we were not in error until now,
            //we set the current time as the start 
            //time of errors
            if err_start_sec == 0 {
              err_start_sec = time.Now().Unix()
            }

            actual_downtime = time.Now().Unix()-err_start_sec
    
            var pgErr *pgconn.PgError
            if errors.As(err, &pgErr) {
              //DEBUG
              //fmt.Println(pgErr.Message) // => syntax error at end of input
              //fmt.Println(pgErr.Code) // => returns the error code, eg 42601
             
              //Printout the most legit error message out to screen
              pgerr(err, pgErr,false)

              match, _ := regexp.MatchString("(select|delete|update|insert)", statement)
              if match {
                errors_count += 1 
              }

              //DEBUG
              //error_log = error_log + "\n[" + time.Now().Format("2006-01-02 15:04:05") + "]" + "[" + pgErr.Code + "] " + pgErr.Message
            }
            previous_stmt_error = true
            previous_loop_err   = true
          } else {
            //the last statement is executed OK

            //we check if the statement is any of SELECT, DELETE, UPDATE, INSERT
            //executed because we only count those in the statements counters
            match, _ := regexp.MatchString("(select|delete|update|insert)", statement)
            if match {
              if err_start_sec != 0 {
                err_stop_sec = time.Now().Unix()
                total_downtime += err_stop_sec - err_start_sec
                err_start_sec = 0
              }
              success_count += 1
            }
          } //           if err != nil { 
        } //for _, statement := range statements {

        //if there were no error executing the WHOLE script.sql (all
        //statements) THEN ONLY we print the update of statements succeeded
        if !previous_stmt_error {
          if previous_loop_err {
            //if the previous loop was in error
            //now it isn't anymore so we can output a message
            //to say everything went back to normal after the last downtime
            previous_loop_err = false
            fmt.Printf(ClearLine);
            fmt.Printf(MoveCursorCol1);
            fmt.Print(string(colorGreen))
            fmt.Printf("Server now running OK after %ds total downtime\n", actual_downtime)
          } 

          // print out results of execution of all statements
          //DEBUG
          //fmt.Printf("\rScript statements succeeded   : |%08d|                             %s\n", success_count, statement)
          fmt.Print(string(colorGreen))
          fmt.Printf(ClearLine);
          fmt.Printf("\rScript statements succeeded   : |%08d|                               ", success_count)
          fmt.Print(string(colorReset)) 
        } else {
          previous_loop_err = true
        }
      } //if !disconnected
    }
  }

	fmt.Println(string(colorReset))

	if connectionInstance.IsClosed() {
		tmpConn, err := pgx.Connect(context.Background(), ReadConfig())
		if err == nil {
			connectionInstance = tmpConn
		}
	}

	err = connectionInstance.Close(context.Background())
  if err != nil {
    exit1("Unable to close connection:\n",err)
	}
  total_exec_time = time.Now().Unix() - total_start_sec

  if total_exec_time == 0 {
    statements_per_sec = success_count 
  } else {
    statements_per_sec = success_count / total_exec_time
  }
 
  //clear current line : shows the previous "running counter"
  fmt.Printf(ClearLine);
  fmt.Printf(MoveCursorCol1);
  fmt.Printf(ClearLine);
  fmt.Printf(MoveCursorCol1);

  // print a Summary
  fmt.Println("=========================================================================")
 fmt.Println("Summary")
  fmt.Println("=========================================================================")
  fmt.Print(string(colorGreen))
  fmt.Printf("\rScript statements commits     : %8d", success_count)
  fmt.Printf(" (statements/second : %4d)\n", statements_per_sec)
 
  if errors_count > 0  {
    fmt.Print(string(colorRed))
    if total_downtime == 0 {
      statements_per_sec = errors_count 
    } else {
      statements_per_sec = errors_count / total_downtime
    }
    fmt.Printf("\rScript statements rollbacks   : %8d", errors_count)
    fmt.Printf(" (statements/second : %4d)\n", statements_per_sec)
  } else {
    fmt.Print(string(colorGreen))
    fmt.Printf("\rScript statements rollbacked  : none")
  }

  fmt.Print(string(colorGreen))
  fmt.Printf("\rTotal exec time               : %8d seconds\n", total_exec_time)
  fmt.Print(string(colorReset))
 
  if !(total_downtime==0) {
    fmt.Print(string(colorRed))
    fmt.Printf("\rTotal downtime                : %8d seconds\n", total_downtime)
    fmt.Print(string(colorReset))
  }

  fmt.Println("=========================================================================")

  //DEBUG
  //if !(error_log=="") {
  //  fmt.Println("Errors log")
  //  fmt.Println("=========================================================================")
  //  fmt.Println(error_log)
  //  fmt.Println("=========================================================================")
  //}
}

// ***** main ******

func main() {

  CheckFlags() 

  fmt.Println(string(colorReset))
  fmt.Printf("%s\n",License_short_notice)

  fmt.Println("=========================================================================")
	fmt.Println("Welcome to ",Version)
  fmt.Println("=========================================================================")

  fmt.Print(string(colorGreen))

  if patroniconfigfilename.set {
	  fmt.Println("About to start in Patroni-monitoring mode")
    fmt.Print(string(colorReset))
    fmt.Println("=========================================================================")
  	fmt.Println("Hit <Enter> to Start")
	  fmt.Println("Hit <Esc> to Exit/Stop anytime after")

	  // Wait for Enter or Esc Key
	  if err := keyboard.Open(); err != nil {
      exit1("Error:\n",err)
	  }

    for {
      _, key, err := keyboard.GetKey()
      if err != nil {
        exit1("Error:\n",err)
      }
      if key == keyboard.KeyEsc {
        break
      } else if key == keyboard.KeyEnter {
        PatroniWatch()
        _ = keyboard.Close()
        break
      }
    }

  } else {
	  fmt.Println("About to start in SQL-loop mode")
    fmt.Print(string(colorReset))
    fmt.Println("=========================================================================")
	  fmt.Println("Hit <Enter> to start")
	  fmt.Println("Hit <Esc> to Exit/Stop anytime after")


	  // Wait for Enter or Esc Key
	  if err := keyboard.Open(); err != nil {
      exit1("Error:\n",err)
	  }

	  for {
		  _, key, err := keyboard.GetKey()
		  if err != nil {
			  //exit1("Error:\n",err)
		  }
		  if key == keyboard.KeyEsc {
			  break
		  } else if key == keyboard.KeyEnter {
   
        if createfilename.set { 
          //user requested to execure a SQL DML/DDL script
          //before looping on the queries of the SQL script 
          //so we do exec it
          ExecCreate()
        }

        if sessiongucsfilename.set { 
          //user requested to throw special SET TRANSACTIONs parameters
          //before looping on the queries of the SQL Script
          //we create a SQL text file to be sent prior executing the main loop
          
          flag.Parse()

          // Open the session_parameters JSON file
          gucs_parameters_file, err := os.Open(sessiongucsfilename.value)
  
          // if we os.Open returns an error then handle it
          if err != nil {
            exit1("Could not open session parmeters GUCs file:\n", err)
          }     
  
          fmt.Println("The following Session Parameters are set:")
  
          // defer the closing of our jsonFile so that we can parse it later on
          defer gucs_parameters_file.Close()
    
          // read our opened xmlFile as a byte array.
          byteValue, _ := ioutil.ReadAll(gucs_parameters_file)
  
          // we initialize our SessionParameters array
          var q SessionParameters
  
          // we unmarshal our byteArray which contains our
          // jsonFile's content into 'sessionparameters' which we defined above
          json.Unmarshal(byteValue, &q)
    
          // we iterate through every SessionParameter within our
          // SessionParametersarray and print out the SessionParameter Type
          // the name and the value
          for i := 0; i < len(q.SessionParameters); i++ {
            //DEBUG 
            //fmt.Println("  Parameter : " + q.SessionParameters[i].Parameter)
            //DEBUG 
            //fmt.Println("  Value   : " + q.SessionParameters[i].Value)
            sessiongucs = sessiongucs + "SET " + q.SessionParameters[i].Parameter + " TO '" + q.SessionParameters[i].Value + "';\n"
            fmt.Println("  ","SET " + q.SessionParameters[i].Parameter + " TO '" + q.SessionParameters[i].Value + "';")
          }
        }

	      fmt.Println("")
        run_simload()
			  break
		  }
	  }
  }
  _ = keyboard.Close()
}
