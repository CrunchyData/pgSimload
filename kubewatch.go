package main

import (
	"fmt"
  "flag"
	"os"
  "os/exec"
  "time"
	"github.com/eiannone/keyboard"
  "encoding/json"
  "strings"
)

var (
  //Kubewatch watcher mode
  kube_watch_timer        int
  kubeconfigfilename      stringFlag
  kubeout                 string = ""
  output                  string = ""
)


type KubeConfig struct {
    Namespace        string
    Watch_timer      int
    Limiter_instance string
    Pod_name         string
    Pod_role         string
    Cluster_name     string
    Node_name        string
    Pod_zone         string
    Pod_status       string
    Master_caption   string
    Replica_caption  string
    Down_caption     string
}

// function ReadKubeConfig() to
// read kubewatch.json to get parameters about Kube monitoring mode this
// function gives values to global variables
func ReadKubeConfig () KubeConfig {

  flag.Parse()
  file, _ := os.Open(kubeconfigfilename.value)
  defer file.Close()
  decoder := json.NewDecoder(file)
  configuration := KubeConfig{}
  err := decoder.Decode(&configuration)
  if err != nil {
    message := "Error while reading the file "
    message += kubeconfigfilename.value +":\n"
    exit1(message,err)
  } 
  return configuration
}


func KubeWatch_k8s(kube_config KubeConfig) {

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

        // variable to handle exec time 
        // and adjust the loop to match
        // user's expectations
        start := time.Now()

        //build the main kube command
        command_args := "kubectl get pods --no-headers"
        command_args += " -n '" + kube_config.Namespace+"'"
        command_args += " -l '" + kube_config.Limiter_instance+"'"
        command_args += " -o custom-columns=\""
        command_args += kube_config.Pod_name+","
        command_args += kube_config.Pod_role+","
        command_args += kube_config.Cluster_name+","
        command_args += kube_config.Node_name+"\""
        command_args += " | sort"
        cmd  := exec.Command("sh", "-c", command_args)

        out, err := cmd.CombinedOutput()

        if err != nil {
          message := "Error executing this command:\n"
          message += "sh -c \"" + command_args + "\"\n" + string(out) + "\n"
          exit1(message,err)
        }

        kubeout = string(out)
  
        //DEBUG
        //fmt.Println("cmd     :"+command_args)
        //fmt.Println("kubeout :"+kubeout)
        //time.Sleep(4000 * time.Millisecond)

        no_cluster := ""

        if ( kubeout == "" ) {
          no_cluster = "+ No PG cluster found in namespace "+kube_config.Namespace
        }

        //divide this output : one record on each '\n' at the end
        lines := strings.Split(kubeout, "\n")

        // Tableau à deux dimensions pour stocker les valeurs
        var records [][]string
 
        //we create an array : each line is a record, put each 
        //value, one per column

        for i := 0; i < len(lines); i++ {
          line := lines[i]
  
          //if line is empty go to the next one
          if line == "" {
            continue
          }
          //remove all spaces and divide 
          columns := strings.Fields(line)
          records = append(records, columns)
        }

        //add ZONE and STATUS to records[][] (for each node)
        //when the infra is setting up, "<none>" can be the poed name
        //(record[3]) so for that special case, the kubect get node will fail
        // 
        // so in the above loop, if "<none>" is pods's name we build an empty
        // output so it works whatever happens..
        //
        for i :=0; i < len(records); i++ {
          record := records[i]

          //(worker)node = record[3]
 
          command_args := "kubectl get node --no-headers " + record[3]
          command_args += " -o custom-columns=\""+kube_config.Pod_zone+","+kube_config.Pod_status+"\""
          cmd  := exec.Command("sh", "-c", command_args)
 
          err_count := 0 // err_count will be units of 1 second

          out, err := cmd.CombinedOutput()
          if err != nil {
            if ( record[3] != "<none>" ) {
              fmt.Println("+ Waiting for pod(s) to be up")
              for err != nil  {
                time.Sleep(1000 * time.Millisecond)
                cmd := exec.Command("sh", "-c", command_args)
                out, err := cmd.CombinedOutput()
                fmt.Printf(".")
                err_count += 1
                if err_count > 60 {
                  message := "Too many failures. Pod(s) took more than 1 minute to be up" 
                  message += "Please carrefully check the following:\n"
                  message += "Error executing this command:\n"
                  message += "sh -c \"" + command_args + "\"\n" + string(out) + "\n"
                  message += "Try to re-run pgSimload once pods are up?"
                  exit1(message,err)
                }
              }
            } else {
              out = []byte("<none> <none>")
            }
          }

          kubeout = string(out)
          lines := strings.Split(kubeout, "\n")

          for j := 0; j < len(lines); j++ {
            line := lines[j]

            //if line is empty go to the next one
            if line == "" {
              continue
            }
            columns := strings.Fields(line)

            //Pod Zone
            records[i] = append(records[i],columns[0])

            //Pod status
            records[i] = append(records[i],columns[1])
          }
        }

        //compute highest lenght of podname
        //to have buttons aligned in the output
        dummy := ""
        for i :=0; i< len(records); i++ {
          record := records[i]
          dummy += record[0]+" "
        }

        longest := LongestOf(dummy)
  
        //create final output to be displayed as a string with \n
        previouscluster := ""
        output = ""
        // for all records 
        for i :=0; i < len(records); i++ {
          record := records[i]
          if (previouscluster != record[2]) {
            output = output + "\n"
            output = output + "+ Cluster : "+ record[2] + "\n"
          }
          
          previouscluster = record[2]
          
          /* Reminder_____________
          Pod name     => record[0] 
          Pod role     => record[1]
          Cluster name => record[2]
          Node name    => record[3]
          Pod's zone   => record[4]
          Pod's status => record[5]
          ______________________ */
 
          role := ""
          button := ""
          zone := ""
          podname := ""
 
          if ( record[5] != "Ready" ) {
            role = " "+kube_config.Down_caption
            button = " 🟥"
          } else if ( record[1] == "master" ) {
            //PGO
            role = " "+kube_config.Master_caption
            button = " 🟩"
          } else if ( record[1] == "primary" ) {
            //CloudNativePG
            role = " "+kube_config.Master_caption
            button = " 🟩"
          } else if ( record[1] == "replica" ) {
            role = " "+kube_config.Replica_caption
            button = " 🟦"
          } else {
            role = " "+kube_config.Down_caption
            button = " 🟥"
          }
  
          if ( record[4] == "<none>") {
            //empty zone
            zone = ""
          } else {
            zone=" ("+record[4]+")"
          }
            
          podname = record[0]

          //line starts with 2 empty spaces
          output += "  " 
          output += PadRight(podname, " ", longest)
          output += zone + button + role + "\n"
        }

        if ( no_cluster != "" ) {
          output = no_cluster 
        }

        if kube_config.Watch_timer >= 1 {

          kube_watch_timer = kube_config.Watch_timer

          // Clears the screen
          fmt.Printf("\x1bc")
          currentTime := time.Now()
    
          fmt.Println("+ Kube-watcher at", currentTime.Format("2006.01.02 15:04:05"))

          //prints out the result
          fmt.Println(output)

          //it took actually that real_exec_time to execute that step in the
          //main loop
          real_exec_time :=  time.Since(start)

          //sleep for a computed time to match kube_config.Watch_timer
          //see ComputedSleep comment for more explanations
          ComputedSleep (real_exec_time, kube_config.Watch_timer)

        } else {

          // Watch_timer is something inferior to 1 : we run once only
  
          // Clears the screen
          fmt.Printf("\x1bc")

          currentTime := time.Now()

          fmt.Println("+ Kube-watcher at", currentTime.Format("2006.01.02 15:04:05"))
 
          //prints out the result 
          fmt.Println(output)

          exit1("Watch_timer in "+kubeconfigfilename.value+" is not >=1 so we ran only once",nil)

        } // if kube_config.Watch_timer > 1
    } //select
  } //for
} // func KubeWatch_k8s()

func KubeWatch() {

  flag.Parse()

  // read the config JSON of the Kube watcher mode
  kube_config := ReadKubeConfig ()

  // try to run kubectl once to insure it is installed
  cmd := exec.Command("kubectl")
  err := cmd.Run()
  if err != nil {
    message := "kubectl is not present on this system. Please install it prior running"
    message = message + "\npgSimload in Kube-watcher mode against a k8s env\n"
    exit1(message,err)
  }

  // launches the main watch loop
  KubeWatch_k8s(kube_config)

}

