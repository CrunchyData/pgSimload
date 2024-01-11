package main

import (
    "context"
    "fmt"
    "time"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
    "os"
    "encoding/json"
    "errors"
)

// PGClientConfig represents the PostgreSQL client configuration.
type PGClientConfig struct {
  Hostname         string  `json:"Hostname"`
  Port             string  `json:"Port"`
  Database         string  `json:"Database"`
  Username         string  `json:"Username"`
  Password         string  `json:"Password"`
  Sslmode          string  `json:"Sslmode"`
  ApplicationName  string  `json:"ApplicationName"`
}


// PGManager represents the PostgreSQL connection manager.
type PGManager struct {
	conn   *pgx.Conn
	Config *PGClientConfig
}

// NewPGManager creates a new PGManager instance with the given configuration.
func NewPGManager(configPath string) (*PGManager, error) {
	config, err := loadConfig(configPath)
	if err != nil {
		return nil, err
	}

	return &PGManager{
		Config: config,
	}, nil
}

func loadConfig(configPath string) (*PGClientConfig, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &PGClientConfig{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// PGConnect establishes a new connection to the PostgreSQL database.
func (pm *PGManager) PGConnect() (*pgx.Conn, error) {
  connStr := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=%s application_name=%s", 
                          pm.Config.Hostname, pm.Config.Port, pm.Config.Database, pm.Config.Username, 
                          pm.Config.Password, pm.Config.Sslmode, pm.Config.ApplicationName)
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		return nil, err
	}

	pm.conn = conn
	return conn, nil
}

// PGReconnectWithTimeout attempts to reconnect to the PostgreSQL database within a specified timeout.
func (pm *PGManager) PGReconnectWithTimeout(timeout time.Duration, err error) error {
  var message string
	startTime := time.Now()

	for time.Since(startTime) < timeout {

    var pgErr *pgconn.PgError

    if errors.As(err, &pgErr) {
      //pgerr(err, pgErr,false)
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

      fmt.Print(string(colorRed))
      fmt.Printf("\r[%s] %s : downtime %s                      ", pgErr.Code,message, time.Since(startTime).Truncate(time.Second).String())
      fmt.Print(string(colorReset))

    } else {
       fmt.Print(string(colorRed))
       fmt.Printf("\rTry reconnecting to PostgreSQL         : downtime %s ", time.Since(startTime).Truncate(time.Second).String())
       fmt.Print(string(colorReset))
    }
 
		//fmt.Println("Attempting to reconnect...")

		err := pm.pgConnectWithRetry()
		if err == nil {
      fmt.Print(string(colorGreen))
			fmt.Printf("\nReconnected successfully after %s downtime", time.Since(startTime).Truncate(time.Second).String())
      fmt.Print(string(colorReset))
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("failed to reconnect within %s", timeout)
}

// pgConnectWithRetry tries to connect to the PostgreSQL database.
func (pm *PGManager) pgConnectWithRetry() error {
	conn, err := pm.PGConnect()
	if err != nil {
		return err
	}

	err = conn.Ping(context.Background())
	if err != nil {
		return err
	}

	pm.conn = conn
	return nil
}
