package main

import (
	"context"
	"fmt"
  "flag"
	"os"
  "time"
  "io/ioutil"
	"github.com/eiannone/keyboard"
  "encoding/json"
)

var (
	success_count      int64 = 0
	errors_count       int64 = 0
  err_start_time     time.Time 
  total_start_time   time.Time 
  total_downtime     time.Duration = 0
)

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


func ExecCreate(pgManager *PGManager) {

  flag.Parse()

  // Open our create.json script with DDLs to run prior
  // to loop on excuting script.sql queries
  create_ddl_file, err := os.Open(createfilename.value)
  
  // if we os.Open returns an error then handle it
  if err != nil {
	  exit1("Could not open DDL script:\n" , err)
  }

  fmt.Println("\nExecuting DDL Script :")

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
    //fmt.Println("DEBUG: SQL Query : " + q.Queries[i].DDL_SQL)
    //fmt.Println("DEBUG: Comment   : " + q.Queries[i].Comment)

    //_, err := connectionInstance.Exec(context.Background(),q.Queries[i].DDL_SQL)
    _, err := pgManager.conn.Exec(context.Background(),q.Queries[i].DDL_SQL)

    if err != nil {
      //connectionInstance.Close(context.Background())
      message := "Something went wrong trying to execute the SQL script\n"
      message = message + createfilename.value + " on the database\n"
      exit1(message,err)
    }

    fmt.Printf("   %q\n",q.Queries[i].Comment)
  }

  fmt.Print(string(colorGreen))
  fmt.Printf("   Script %q successfully executed !\n",createfilename.value)
  fmt.Print(string(colorReset))
}

func SetSessionParameters(pgManager *PGManager) {

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
    sessiongucs = sessiongucs + "SET " + q.SessionParameters[i].Parameter + " TO '" + q.SessionParameters[i].Value + "';\n"
    fmt.Println("  ","SET " + q.SessionParameters[i].Parameter + " TO '" + q.SessionParameters[i].Value + "';")
  }

  //DEBUG
  //fmt.Println("Session GUCS",sessiongucs)
  _, err = pgManager.conn.Exec(context.Background(), sessiongucs)

  if err != nil {
    exit1("Error while trying to set session parameters as described in -session_parameters " + sessiongucsfilename.value+"\n",err)
  } else {
    fmt.Print(string(colorGreen))
    fmt.Printf("   Session parameters applied to the PG session !\n")
    fmt.Print(string(colorReset))
  }
}


func do_sqlloop(pgManager *PGManager) {

  // read script.sql
  script_file, err := ioutil.ReadFile(scriptfilename.value)

  if err != nil {
    exit1("Could not read script file:\n" , err)
  }

  statements := string(script_file)

  //DEBUG
  //exit1(string(script_file),nil)

  //test script once to ensure there's no errors in it
  bad_script := 0
 
  // Exec executes SQL via the PostgreSQL simple query protocol. SQL may contain multiple queries. Execution is
  // implicitly wrapped in a transaction unless a transaction is already in progress or SQL contains transaction control
  // statements. 
  // see https://github.com/jackc/pgx/blob/master/pgconn/pgconn.go#L1047C1-L1049C15
  _, err = pgManager.conn.Exec(context.Background(), statements)

  if err != nil { 
    bad_script += 1
  }

  if bad_script > 0 {
    message := "Execution of SQL script in "+scriptfilename.value+" returns errors"
    message = message + "\nPlease correct the errors prior running pgSimload."
    exit1(message,nil)
  } else {
	  fmt.Println()
    fmt.Printf("Now entering the main loop, executing script %q\n",scriptfilename.value) 
  }

  //store Time when we started the loop
  total_start_time = time.Now()
	
  // MAIN loop on SQL content of script.sql
	// This is to be able to stop the loop on <Esc> key
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

      _, err := pgManager.conn.Exec(context.Background(), statements)

      if err != nil { 
        //the last execution of the script is in error

        //if we were not in error until now,
        //we set the current time as the start 
        //time of errors
        if err_start_time.IsZero() {
          err_start_time = time.Now()
        }

        errors_count += 1 

        //DEBUG
        //fmt.Println("DEBUG: errors_count: "+ string(errors_count))
        //fmt.Println("DEBUG: statements  : "+ statements)

        //we may have been connected ? Let's ping the server
        //if we don't have an answer, try to reconnect instead

        //test connection
        //err := pgManager.conn.Ping(context.Background())
        err = pgManager.conn.Ping(context.Background())
        if err != nil {
          err := pgManager.PGReconnectWithTimeout(pgReconnectTimeout,err)
          if err != nil {
            exit1("\nUnable to reconnect to PostgreSQL:\n", err)
          }
        }
      } else {

        //the last execution of the script is OK

        if ! err_start_time.IsZero() {
          total_downtime += time.Since(err_start_time)
          err_start_time = time.Time{}
        } else {
          success_count += 1
        }

        fmt.Print(string(colorGreen))
        fmt.Printf(ClearLine);
        fmt.Printf("\rScript executions succeeded : %10d                               ", success_count)
        fmt.Print(string(colorReset)) 
      }
    }
  }

  // end of the main SQL Loop, time to compute times, etc..
  // and print the summary before exiting
	fmt.Println(string(colorReset))

  total_exec_time := time.Since(total_start_time)

  var statements_per_sec float64

  if total_exec_time < 1 {
    statements_per_sec =  float64(success_count)
  } else {
    statements_per_sec = float64(success_count) / total_exec_time.Seconds()
  }
 
  //clear current line : shows the previous "running counter"
  //fmt.Printf(MoveCursorCol1);
  //fmt.Printf(ClearLine);

  // print a Summary
  fmt.Println("=========================================================================")
  fmt.Println("Summary")
  fmt.Println("=========================================================================")
  fmt.Print(string(colorGreen))
  //fmt.Printf("\rScript statements commits   : %8d", success_count)
  fmt.Printf("\rScript executions succeeded : %10d", success_count)

  fmt.Printf(" (%.3f scripts/second)\n", statements_per_sec)
 
  if errors_count > 0  {
    fmt.Print(string(colorRed))
    if total_downtime == 0 {
      statements_per_sec = float64(errors_count)
    } else {
      statements_per_sec = float64(errors_count) / total_downtime.Seconds()
    }
    fmt.Printf("\rScript executions rollbacks : %10d", errors_count)
    fmt.Printf(" (%.3f scripts/second)\n", statements_per_sec)
  } else {
    fmt.Print(string(colorGreen))
    fmt.Printf("\rScript scripts rollbacked: none")
  }

  //print out total exec time in Milliseconds if < 10 min total time
  //print out total exec time in Seconds otherwise
  fmt.Print(string(colorGreen))
  if total_exec_time.Truncate(time.Minute) < 10 {
    fmt.Printf("\rTotal exec time             : %10s\n", total_exec_time.Truncate(time.Millisecond).String())
  } else {
    fmt.Printf("\rTotal exec time             : %10s\n", total_exec_time.Truncate(time.Second).String())
  }
  fmt.Print(string(colorReset))
 
  if !(total_downtime==0) {
    fmt.Print(string(colorRed))
    fmt.Printf("\rTotal real downtime         : %10s\n", total_downtime.Truncate(time.Millisecond).String())
    fmt.Print(string(colorReset))
  }

  fmt.Println("=========================================================================")

}

func SQLLoop () {

  // PG connex needed
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
    //we won't try to reconnect here since the loop 
    //did not started yet
    exit1("Failed to connect to PostgreSQL:\n", err)
  }
  defer conn.Close(context.Background())

  if sessiongucsfilename.set {
    SetSessionParameters(pgManager)
  }

  if createfilename.set {
    //user requested to execure a SQL DML/DDL script
    //before looping on the queries of the SQL script 
    //so we do exec it
    ExecCreate(pgManager)
  }
  
  do_sqlloop(pgManager)

}

