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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
  "encoding/json"
  "regexp"
  "errors"
)

var (
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


func sqlloop() {

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
