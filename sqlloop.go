package main

import (
	"context"
	"fmt"
  "flag"
	"os"
  "time"
	"github.com/eiannone/keyboard"
  "encoding/json"
  "sync"
  "math/rand"
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

  fmt.Println("Executing DDL Script :")

  // defer the closing of our jsonFile so that we can parse it later on
  defer create_ddl_file.Close()

  // get the file size
  create_ddl_file_stat, err := create_ddl_file.Stat()
  
  if err != nil {
		fmt.Println("Error getting ddl script size information:", err)
	  exit1("Error getting file size information:\n" , err)
	}

  // read the file as a byte slice (array)
  byteValue := make([]byte, create_ddl_file_stat.Size())

	_, err = create_ddl_file.Read(byteValue)
	if err != nil {
    exit1("Error reading ddl script file:\n", err)
	}

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

    _, err := pgManager.conn.Exec(context.Background(),q.Queries[i].DDL_SQL)

    if err != nil {
      //connectionInstance.Close(context.Background())
      message := "Something went wrong trying to execute the SQL script\n"
      message += createfilename.value + " on the database\n"
      exit1(message,err)
    }

    fmt.Printf("   %q\n",q.Queries[i].Comment)
  }

  fmt.Print(string(colorGreen))
  fmt.Printf("   Script %q successfully executed !\n",createfilename.value)
  fmt.Print(string(colorReset))
  fmt.Println()
}

func SetSessionParameters(pgManager *PGManager, client_id int) {

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

  // defer the closing of our jsonFile so that we can parse it later on
  defer gucs_parameters_file.Close()

  // get the file size
  gucs_parameters_file_stat, err := gucs_parameters_file.Stat()

  if err != nil {
    exit1("Error getting GUCS parameters file size information:\n" , err)
  }

  // read the file as a byte slice (array)
  byteValue := make([]byte, gucs_parameters_file_stat.Size())

  _, err = gucs_parameters_file.Read(byteValue)
  if err != nil {
    exit1("Error reading GUCS parameters file:\n", err)
  }

  if client_id == 1 {
    fmt.Printf("The following session parameters are applied")

    if exec_clients > 1 {
      fmt.Printf(" to all %d clients:\n", exec_clients)
    } else {
      fmt.Printf(":\n")
    }
  }

  // we initialize our SessionParameters GUCS parameters  q SessionParameters
  var q SessionParameters

  // we unmarshal our byteArray which contains our
  // jsonFile's content into 'sessionparameters' which we defined above
  json.Unmarshal(byteValue, &q)

  // we iterate through every SessionParameter within our
  // SessionParametersarray and print out the SessionParameter Type
  // the name and the value
  for i := 0; i < len(q.SessionParameters); i++ {
    sessiongucs = sessiongucs + "SET " + q.SessionParameters[i].Parameter + " TO '" + q.SessionParameters[i].Value + "';\n"
    if client_id == 1 {
      fmt.Println("  ","SET " + q.SessionParameters[i].Parameter + " TO '" + q.SessionParameters[i].Value + "';")
    }
  }

  if client_id == 1 {
    fmt.Println()
  } 

  //DEBUG
  //fmt.Println("Session GUCS",sessiongucs)
  _, err = pgManager.conn.Exec(context.Background(), sessiongucs)

  if err != nil {
    exit1("Error while trying to set session parameters as described in -session_parameters " + sessiongucsfilename.value+"\n",err)
  } 
}

func do_sqlloop() {

  // read script.sql
  script_file, err := os.ReadFile(scriptfilename.value)

  if err != nil {
    exit1("Could not read script file:\n" , err)
  }
 
  statements := string(script_file)

  //test script once to ensure there's no errors in it
  bad_script := 0

  pgManager, err := NewPGManager(configfilename.value)
  if err != nil {
    exit1("Failed to create PGManager:\n", err)
  }

  // Initial connection
  conn, err := pgManager.PGConnect()
  if err != nil {
    // we won't try to reconnect here since the loop 
    // did not started yet
    exit1("Failed to connect to PostgreSQL:\n", err)
  }

  defer conn.Close(context.Background())

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
    message += "\nPlease correct the errors prior running pgSimload."
    exit1(message,nil)
  } else {

    fmt.Printf("Now entering the main loop\n\n")

    fmt.Println("Executing script", scriptfilename.value)
  
    if (sleep_time > 0 && rsleep_time >0) {
      fmt.Printf("each %q ", sleep_time) 
      fmt.Printf("plus a an added random time of maximum %q\n",rsleep_time)
    } else if sleep_time > 0 {
      fmt.Printf("each %q\n", sleep_time)
    } else if rsleep_time > 0 {
      fmt.Printf("with a maximum random sleep time of %q between iterations\n", rsleep_time)
    } else {
      fmt.Printf("as fast as possible\n")
    }
    
    fmt.Println()

    if exec_loops !=0 || exec_time !=0 {
      fmt.Println("Number of loops will be limited:")

      if exec_loops != 0 {
        fmt.Printf("  %d executions\n", exec_loops) 
      } 

      if exec_time != 0 {
        if exec_loops !=0 {
          fmt.Printf(" or\n  %q maximum duration\n", exec_time)
          fmt.Printf("  (whichever happens first)\n")
        } else {
          fmt.Printf("  %q maximum duration\n", exec_time)
        }
      } 

      fmt.Println()
    }

    if exec_clients != 1 {
      fmt.Printf("\nExecuting the loop with %d concurrent clients\n\n", exec_clients)
    }

  }

  conn.Close(context.Background())

  //store Time when we started the loop
  total_start_time = time.Now()

  var wg sync.WaitGroup

  // MAIN loop on SQL content of script.sql
	// This is to be able to stop the loop on <Esc> key
  stopCh    := make(chan bool)
	
	go func() {

	  for {
		  _, key, err := keyboard.GetKey()
			if err != nil {
				//exit1("Error:\n",err)
		  }
			if key == keyboard.KeyEsc {
        for i=1; i < exec_clients+1; i++ {
				  stopCh <- true
        }
		  }
	  }
  }()

  //if user has set a --time "duration" parameter we start a "timer"
  //of that amount of time. This one will send "true" to stopCh once 
  //dead so it will break the loop  
  //basically, it's just to Sleep for exec_time duration..
  go func() {
    if exec_time != 0 {
      time.Sleep(exec_time)
      for i=1; i < exec_clients+1; i++ {
			  stopCh <- true
      }
    }    
  }()


  // spawn as many clients as the user wants
  for i=1; i < exec_clients+1; i++ { 
    //DEBUG
    //fmt.Printf("DEBUG: Spawn do_sqlloop(%d)\n", i)
    wg.Add(1)

    go func(client_id int) {
  
      defer wg.Done()

      pgManager, err := NewPGManager(configfilename.value)
      if err != nil {
        exit1("Failed to create PGManager:\n", err)
      }

      // Initial connection
      conn, err := pgManager.PGConnect()
      if err != nil {
        // we won't try to reconnect here since the loop 
        // did not started yet
        exit1("\nFailed to connect to PostgreSQL:\n", err)
      }

      if sessiongucsfilename.set {
        SetSessionParameters(pgManager, client_id)
      }

      defer conn.Close(context.Background())

    loop:
     	for {
		    select {
		    case stop := <-stopCh:
			    if stop {
				    break loop
            return
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

            //we may have been connected ? Let's ping the server
            //if we don't have an answer, try to reconnect instead

            //test connection
            //err := pgManager.conn.Ping(context.Background())
            err = pgManager.conn.Ping(context.Background())
            if err != nil {
              err := pgManager.PGReconnectWithTimeout(pgReconnectTimeout,err)
              if err != nil {
                stopCh <- true
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

            if exec_clients != 1 {
               fmt.Printf("\rScript executions succeeded : %10d (%d per client)", success_count, success_count/int64(exec_clients))
            } else {
               fmt.Printf("\rScript executions succeeded : %10d ", success_count)
            }
            fmt.Print(string(colorReset)) 

            if (exec_loops !=0) && (success_count >= exec_loops) {
              break loop;
              stopCh <- true
            }

            if sleep_time > 0 {
              time.Sleep(sleep_time)
            } 

            if rsleep_time > 0 {

              // seed the random number gen
	            rand.Seed(time.Now().UnixNano())
  
              //gen a random duration between 0 and rsleep_time
              random_sleep_duration := time.Duration(rand.Int63n(int64(rsleep_time)))

              time.Sleep(random_sleep_duration)
            } 

          }
        }
      }    // loop: for {...}
    }(i)   // go func(client_id int){...}(i)
  }        // for i=1; i < exec_clients+1; i++ {

  wg.Wait() // Wait for all goroutines to finish

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
 
  //DEBUG 
  //fmt.Println("DEBUG: Created a new PGManager")

  // Initial connection
  conn, err := pgManager.PGConnect()
  if err != nil {
    // we won't try to reconnect here since the loop 
    // did not started yet
    exit1("Failed to connect to PostgreSQL:\n", err)
  }

  //defer conn.Close(context.Background())

  //fmt.Println("DEBUG: Connected to PG!")

  // time to clear the screen to remove the licence
  // etc...
  // Clears the screen
  fmt.Printf("\x1bc")

  currentTime := time.Now()
  fmt.Println("+ SQL-Loop at", currentTime.Format("2006.01.02 15:04:05"))
  fmt.Println()

  if createfilename.set {
    //user requested to execure a SQL DML/DDL script
    //before looping on the queries of the SQL script 
    //so we do exec it
    ExecCreate(pgManager)
  }

  conn.Close(context.Background())
 
  do_sqlloop()

}

