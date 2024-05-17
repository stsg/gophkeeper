// Package main contains all application logic
package main

import (
    "context"
    "errors"
    "fmt"
    "log"
    "os"
    "os/signal"
    "strings"
    "syscall"
    "time"

    "github.com/fatih/color"
    "github.com/go-pkgz/lgr"
    "github.com/umputun/go-flags"

    "github.com/stsg/gophkeeper/app/config"
    "github.com/stsg/gophkeeper/app/server"
    "github.com/stsg/gophkeeper/app/status"
)

var revision string

var opts struct {
    Config  string        `short:"f" long:"config" env:"CONFIG" description:"config file"`
    Listen  string        `short:"l" long:"listen" env:"LISTEN" default:"localhost:8080" description:"listen address"`
    Volumes []string      `short:"v" long:"volumes" env:"VOLUMES" default:"root:/" env-delim:"," description:"comma separated list of volumes to monitor"`
    Timeout time.Duration `short:"t" long:"timeout" env:"TIMEOUT" default:"10s" description:"connection timeout"`
    Dbg     bool          `long:"dbg" env:"DEBUG" description:"show debug info"`
}

func main() {
    fmt.Printf("gophkeeper %s\n", revision)

    p := flags.NewParser(&opts, flags.PassDoubleDash|flags.HelpFlag)
    if _, err := p.Parse(); err != nil {
        if err.(*flags.Error).Type != flags.ErrHelp {
            fmt.Printf("%s\n", err)
            os.Exit(1)
        }
        p.WriteHelp(os.Stderr)
        os.Exit(2)
    }
    setupLog(opts.Dbg)

    ctx, cancel := context.WithCancel(context.Background())
    go func() {
        if x := recover(); x != nil {
            log.Printf("[WARN] runtime panic:\n%v", x)
            panic(x)
        }

        // catch signal for graceful shutdown
        stop := make(chan os.Signal, 1)
        signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
        <-stop
        log.Printf("[WARN] shutdown by signal")
        cancel()
    }()

    var conf *config.Parameters

    if opts.Config != "" {
        var err error
        conf, err := config.New(opts.Config)
        if err != nil {
            log.Printf("[ERROR] can't load config: %s", err)
        }
        log.Printf("[DEBUG] loaded config: %s", conf.String())
    }

    vols, err := parseVolumes([]string{"vol1:/"}, conf)
    if err != nil {
        log.Fatalf("[ERROR] %s", err)
    }

    srv := server.Rest{
        Listen:  opts.Listen,
        Version: revision,
        Config:  conf,
        Status: &status.Host{
            Volumes: vols,
        },
    }

    if err := srv.Run(ctx); err != nil && err.Error() != "http: Server closed" {
        log.Fatalf("[ERROR] %s", err)
    }

}

func parseVolumes(volumes []string, conf *config.Parameters) ([]status.Volume, error) {
    res := []status.Volume{}

    // load from config if present and volumes provided
    if conf != nil && len(conf.Volumes) > 0 {
        for _, v := range conf.Volumes {
            res = append(res, status.Volume{Name: v.Name, Path: v.Path})
        }
    }

    if len(volumes) > 0 {
        res = []status.Volume{}
        for _, v := range volumes {
            parts := strings.SplitN(v, ":", 2)
            if len(parts) != 2 {
                return nil, errors.New("invalid volume format, should be <name>:<path>")
            }
            res = append(res, status.Volume{Name: parts[0], Path: parts[1]})
        }
    }

    log.Printf("[DEBUG] volumes: %+v", res)
    return res, nil
}

// setupLog sets up the logger with the given debug mode.
//
// It takes a boolean parameter dbg and does not return anything.
func setupLog(dbg bool) {
    logOpts := []lgr.Option{lgr.Msec, lgr.LevelBraces, lgr.StackTraceOnError}
    if dbg {
        logOpts = []lgr.Option{lgr.Debug, lgr.CallerFile, lgr.CallerFunc, lgr.Msec, lgr.LevelBraces, lgr.StackTraceOnError}
    }

    colorizer := lgr.Mapper{
        ErrorFunc:  func(s string) string { return color.New(color.FgHiRed).Sprint(s) },
        WarnFunc:   func(s string) string { return color.New(color.FgRed).Sprint(s) },
        InfoFunc:   func(s string) string { return color.New(color.FgYellow).Sprint(s) },
        DebugFunc:  func(s string) string { return color.New(color.FgWhite).Sprint(s) },
        CallerFunc: func(s string) string { return color.New(color.FgBlue).Sprint(s) },
        TimeFunc:   func(s string) string { return color.New(color.FgCyan).Sprint(s) },
    }
    logOpts = append(logOpts, lgr.Map(colorizer))

    lgr.SetupStdLogger(logOpts...)
    lgr.Setup(logOpts...)
}
