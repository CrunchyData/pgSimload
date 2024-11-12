package main

import (
	"fmt"
  "flag"
	"os"
	"github.com/eiannone/keyboard"
  "github.com/MakeNowJust/heredoc"
  "time"
  "strings"
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
	i                         int = 0
  configfilename            stringFlag
  createfilename            stringFlag
  scriptfilename            stringFlag
  exec_clients              int = 1
  exec_loops                int64 = 0
  exec_time                 time.Duration
  sleep_time                time.Duration
  rsleep_time               time.Duration
  silent_start              bool 

  Version      = "v.1.4.2"
  Release_date = "November, 12nd 2024"

  License = heredoc.Doc(`
**The PostgreSQL License**

Copyright (c) 2022-2024, Crunchy Data Solutions, Inc.

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

  Contact = "You can contact programmer at : Jean-Paul Argudo <jean-paul.argudo@crunchydata.com>\nProject is hosted on https://github.com/CrunchyData/pgSimload"
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


//String Flags
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

// strings function to pad left, right, given a size of biggest element in the
// list
func LongestOf(s string) int {
    length := 0
    for _, word := range strings.Split(s, " ") {
        if len(word) > length {
            length = len(word)
        } 
    }       
    return length
}         
            
func PadRight(str, pad string, lenght int) string {
  for {   
    str += pad
    if len(str) > lenght {
      return str[0:lenght]
    }       
  }       
}           
          
func PadLeft(str, pad string, lenght int) string {
  for {   
    str = pad + str
    if len(str) > lenght {
      return str[0:lenght] 
    }
  }
}


func init() {
  flag.Var(&configfilename,        "config",     "JSON config filename")
  flag.Var(&createfilename,        "create",     "JSON create filename")
  flag.Var(&scriptfilename,        "script",     "SQL script filename")
  flag.Var(&sessiongucsfilename,   "session_parameters", "JSON session gucs filename")
  flag.Var(&patroniconfigfilename, "patroni",    "JSON Patroni watcher mode config filename")
  flag.Var(&kubeconfigfilename   , "kube" ,      "JSON Kube watcher mode config filename")
  flag.Var(&gathergucsfilename   , "create_gucs_template", "outputs to that JSON filename") 
  flag.IntVar(&exec_clients,       "clients", 1, "number of SQL-Loop to execute concurrently") 
  flag.Int64Var(&exec_loops,       "loops",   0, "number of SQL-Loop to execute") 
  flag.DurationVar(&exec_time,     "time" ,   0, "duration of SQL-Loop execution")
  flag.DurationVar(&sleep_time,    "sleep",   0, "sleep duration between iterations in SQL-Loop")
  flag.DurationVar(&rsleep_time,   "rsleep",  0, "maximum random sleep duration between iterations in SQL-Loop")

}

//function to check flags passed with --flag value
//upon execution of the tool
func CheckFlags () {
 
  help    := flag.Bool("help", false, "display some help")
  version := flag.Bool("version", false, "display version")
  license := flag.Bool("license", false, "display license")
  contact := flag.Bool("contact", false, "display where to contact programmers")  
  silent  := flag.Bool("silent",  false, "don't display start banner")

  flag.Parse()

  if *version {
		_ = keyboard.Close()
    fmt.Printf("pgSimload %s - %s\n",Version,Release_date);
    os.Exit(0);
  }
   
  if *license {
		_ = keyboard.Close()
    fmt.Printf("pgSimload %s - %s is licensed under \n",Version,Release_date);
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
    fmt.Println("Please read documentation in doc/\nAlternatively, run with -h to show all possible parameters.");
    os.Exit(0);
  } 

  if *silent {
    silent_start = true
  } 

  if gathergucsfilename.set {
    if !configfilename.set {
      message := "To create a template JSON file to be used in -session_parameters\n"
      message += "You actually have to use a -config config.json in conjunction with it\n"
      exit1(message,nil)
    } else {
      gatherGucs()
      _ = keyboard.Close()
      os.Exit(0)
    }
  } 

  if !patroniconfigfilename.set && !kubeconfigfilename.set {
  
    //not in Patroni-Watcher mode.. 
    //nor Kube-Watcher mode...
    
    //checking at least a config AND a script is present because that's
    //SQL-Loop then... (default)

    if !(scriptfilename.set || configfilename.set) {
      message := "Please read documentation in doc/ since parameters have to be passed"
      message += "\nAlternatively, run with -h to show all possible parameters"
      exit1(message,nil)
    }
 
    if !scriptfilename.set {
       if configfilename.set {
         message := "You have set a config filename but no script filename (--script <SQL file>) !"
         message += "\nPlease read documentation in doc/ and/or run with -h"
         exit1(message,nil) 
       }
    }
  
    if !configfilename.set {
      if scriptfilename.set {
        message := "You have set a script filename but no config filename (--config <JSON file>) !"
        message += "\nPlease read documentation in doc/ and/or run with -h"
        exit1(message,nil)
      }
    }

  }

}

func start_banner (mode string) {
  fmt.Println(string(colorReset))
  fmt.Printf("%s\n",License_short_notice)
  fmt.Println("=========================================================================")
  fmt.Println("Welcome to pgSimload ",Version)
  fmt.Println("=========================================================================")
  fmt.Print(string(colorGreen))
	fmt.Println("About to start in "+mode+" mode")
  fmt.Print(string(colorReset))
  fmt.Println("=========================================================================")
  fmt.Println("Hit <Enter> to Start")
	fmt.Println("Hit <Esc> to Exit/Stop anytime after")
}


// ***** main ******

func main() {

  CheckFlags() 

  //open keyboard
	if err := keyboard.Open(); err != nil {
    exit1("Error:\n",err)
	}

  if patroniconfigfilename.set {

    if silent_start {
      
      //silent start
      PatroniWatch()
    
    } else {
 
      //adds info to start banner: we're starting in Patroni-monitoring mode
      start_banner("Patroni-Watcher")

      //Wait for key
      // - ESC will cancel the execution
      // - ENTER will start the execution
      for {
        _, key, err := keyboard.GetKey()
        if err != nil {
          exit1("Error:\n",err)
        }
        if key == keyboard.KeyEsc {
          break
        } else if key == keyboard.KeyEnter {
          PatroniWatch()
          break
        }
      }
    }

  } else if kubeconfigfilename.set {


    if silent_start {

      //silient start 
      KubeWatch()

    } else {
 
      //adds info to start banner: we're starting in Kube-watcher mode
      start_banner("Kube-Watcher")
  
      //Wait for key
      // - ESC will cancel the execution
      // - ENTER will start the execution
      for {
        _, key, err := keyboard.GetKey()
        if err != nil {
          exit1("Error:\n",err)
        }
        if key == keyboard.KeyEsc {
          break
        } else if key == keyboard.KeyEnter {
          KubeWatch()
          break
        }
      }
    }

  } else {

    if silent_start {
      
      //silent start
      SQLLoop()

    } else {

      //adds info to start banner: we're starting in SQL-loop  mode
      start_banner("SQL-Loop")

      //Wait for key
      // - ESC will cancel the execution
      // - ENTER will start the execution
	    for {
	  	  _, key, err := keyboard.GetKey()
	  	  if err != nil {
  			  exit1("Error:\n",err)
	  	  }
		    if key == keyboard.KeyEsc {
		  	  break
		    } else if key == keyboard.KeyEnter {
          SQLLoop()
		  	  break
		    }
	    }  
    }
  }
  _ = keyboard.Close()
}
