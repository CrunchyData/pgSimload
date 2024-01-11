package main

import (
	"context"
	"fmt"
  "flag"
	"os"
  "time"
  "io/ioutil"
  "strings"
	"github.com/eiannone/keyboard"
  "encoding/json"
  "regexp"
)

var (
	success_count      int64 = 0
	errors_count       int64 = 0
  err_start_time     time.Time 
  err_stop_time      time.Time 
  total_start_time   time.Time 
  actual_downtime    time.Duration
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

  statements := strings.Split(string(script_file), ";\n")

  // last element is an empty line
  statements = statements[:len(statements)-1]

	fmt.Println()
	//fmt.Println("Statements for main loop loaded!")
  fmt.Printf("Now entering the main loop, executing script %q\n",scriptfilename.value) 

  //store Time when we started the loop
  total_start_time = time.Now()
	
  // MAIN loop on your command(s) in script.sql
	// This is to be able to stop the loop on <Esc>
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

      var previous_stmt_error bool = false
      var previous_loop_error bool = false

      for _, statement := range statements {

        // we first lowercase the statement which facilitates a later search
        // for insert, delete, update and select in a later regexp
        // because we only count those statemets in the counters
        statement = strings.ToLower(statement)

        //we execute each statement, one by one
        _, err := pgManager.conn.Exec(context.Background(), statement)

        if err != nil { 
          //the last statement is in error

          //if we were not in error until now,
          //we set the current time as the start 
          //time of errors
          if err_start_time.IsZero() {
            err_start_time = time.Now()
          }

          actual_downtime = time.Since(err_start_time)
    
          match, _ := regexp.MatchString("(select|delete|update|insert)", statement)
          if match {
            errors_count += 1 
          }

          previous_stmt_error = true
          previous_loop_error = true

        } else {
          //the last statement is executed OK

          //we check if the statement is any of SELECT, DELETE, UPDATE, INSERT
          //executed because we only count those in the statements counters
          match, _ := regexp.MatchString("(select|delete|update|insert)", statement)
          if match {
            if ! err_start_time.IsZero() {
              err_stop_time = time.Now()
              //total_downtime += err_stop_sec - err_start_sec
              total_downtime += time.Since(err_start_time)
              //err_start_time = nil
              err_start_time = time.Time{}
            } else {
              success_count += 1
            }
          }
          previous_stmt_error = false
        }
      } //for _, statement := range statements {

      //test connection
      err := pgManager.conn.Ping(context.Background())
      if err != nil {
        err := pgManager.PGReconnectWithTimeout(pgReconnectTimeout,err)
        if err != nil {
          exit1("Failed to reconnect:\n", err)
        }
      }

      //if there were no error executing the WHOLE script.sql (all
      //statements) THEN ONLY we print the update of statements succeeded
      if !previous_stmt_error {
        if previous_loop_error {
          //if the previous loop was in error
          //now it isn't anymore so we can output a message
          //to say everything went back to normal after the last downtime
          previous_loop_error = false
          actual_downtime = 0
        } 

        // print out results of execution of all statements
        //DEBUG
        //fmt.Printf("\rScript statements succeeded   : |%08d|                             %s\n", success_count, statement)
        fmt.Print(string(colorGreen))
        fmt.Printf(ClearLine);
        fmt.Printf("\rScript statements succeeded   : |%08d|                               ", success_count)
        fmt.Print(string(colorReset)) 
      } else {
        previous_loop_error = true
      }
    }
  }

	fmt.Println(string(colorReset))

  total_exec_time := time.Since(total_start_time)

  var statements_per_sec int64

  if total_exec_time == 0 {
    statements_per_sec = success_count
  } else {
    statements_per_sec = success_count / int64(total_exec_time.Seconds())
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
      statements_per_sec = errors_count / int64(total_downtime.Seconds())
    }
    fmt.Printf("\rScript statements rollbacks   : %8d", errors_count)
    fmt.Printf(" (statements/second : %4d)\n", statements_per_sec)
  } else {
    fmt.Print(string(colorGreen))
    fmt.Printf("\rScript statements rollbacked  : none")
  }

  fmt.Print(string(colorGreen))
  fmt.Printf("\rTotal exec time               : %8s\n", total_exec_time.Truncate(time.Second).String())
  fmt.Print(string(colorReset))
 
  if !(total_downtime==0) {
    fmt.Print(string(colorRed))
    fmt.Printf("\rTotal downtime                : %8s\n", total_downtime.Truncate(time.Second).String())
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

