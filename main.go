package main

import (
	"fmt"
  "flag"
	"os"
	"github.com/eiannone/keyboard"
  "encoding/json"
  "github.com/MakeNowJust/heredoc"
)
//"io/ioutil"

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
  configfilename            stringFlag
  createfilename            stringFlag
  scriptfilename            stringFlag

  Version = "pgSimLoad v.1.0.2 - January 11th 2023"

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

func start_banner (mode string) {

  switch (mode) {
    case "start":
      fmt.Println(string(colorReset))
      fmt.Printf("%s\n",License_short_notice)
      fmt.Println("=========================================================================")
      fmt.Println("Welcome to ",Version)
      fmt.Println("=========================================================================")
      fmt.Print(string(colorGreen))
    case "Patroni-monitoring","SQL-loop": 
	    fmt.Println("About to start in "+mode+" mode")
      fmt.Print(string(colorReset))
      fmt.Println("=========================================================================")
  	  fmt.Println("Hit <Enter> to Start")
	    fmt.Println("Hit <Esc> to Exit/Stop anytime after")
  }
}


// ***** main ******

func main() {

  CheckFlags() 

  //prints out the start banner
  start_banner("start")

  if patroniconfigfilename.set {
 
    //adds info to start banner: we're starting in Patroni-monitoring mode
    start_banner("Patroni-monitoring")

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

    //adds info to start banner: we're starting in SQL-loop  mode
    start_banner("SQL-loop")

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
   
	      fmt.Println("")
        SQLLoop()
			  break
		  }
	  }
  }
  _ = keyboard.Close()
}
