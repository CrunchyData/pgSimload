package main

import (
  "context"
	"fmt"
  "flag"
  "io/ioutil"
  "github.com/MakeNowJust/heredoc"
)

var (
  //Session Parameters
  sessiongucs               string = ""
  sessiongucsfilename       stringFlag
  gathergucsfilename        stringFlag
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
)

// Function to gather modifiable per-session GUCS, included contexts
// 'user' and 'superuser', because we don't know in advance what will
// be the user defined in the config.json file... Will be up to the
// user to try to modify things in the session_parameter.json file 
// that this function will write into (gathergucsfilename.value) 
func gatherGucs () {

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

  flag.Parse()

  var file string = gather_gucs_file_header

  rows, _ := conn.Query(context.Background(), gather_gucs_query)

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

  fmt.Print(string(colorGreen))
  fmt.Println("Session parameters template file created !")
  fmt.Print(string(colorReset))
  fmt.Println("You can now edit "+gathergucsfilename.value+" to suit your needs")
  fmt.Println("to be used afterwards with -session_parameters in SQL-loop mode")

}
