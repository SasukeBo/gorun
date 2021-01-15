package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
)

var usageStr = `
Usage: gorun [options] <subject>

Example: gorun -id gorun main.go

Options:
	-ip, --apollo_ip	<url>				APOLLO API server URL
	-c,  --cluster   	<cluster name>		APOLLO cluster name
	-id, --app_id		<app id>			APOLLO app id
	-k,  --key			<access key>		APOLLO access key
	-r,  --registry		<service registry>	Micro service registry
	-t,  --test			 					Go test
`

const (
	defaultApolloIP = "apollo.api.test.thingyouwe.com"
	defaultCluster  = "wb_local"
	defaultRegistry = "etcd"
)

func usage() {
	fmt.Printf("%s\n", usageStr)
	os.Exit(0)
}

func main() {
	var (
		apolloIP string
		cluster  string
		appID    string
		key      string
		registry string
		test     bool
		cmd      *exec.Cmd
	)

	flag.StringVar(&apolloIP, "ip", defaultApolloIP, "The Apollo config api server URL")
	flag.StringVar(&apolloIP, "apollo_ip", defaultApolloIP, "The Apollo config api server URL")
	flag.StringVar(&cluster, "c", defaultCluster, "The Apollo cluster name")
	flag.StringVar(&cluster, "cluster", defaultCluster, "The Apollo cluster name")
	flag.StringVar(&appID, "id", "", "The Apollo app id")
	flag.StringVar(&appID, "app_id", "", "The Apollo app id")
	flag.StringVar(&key, "k", "", "The Apollo access key")
	flag.StringVar(&key, "key", "", "The Apollo access key")
	flag.StringVar(&registry, "r", defaultRegistry, "The micro service registry")
	flag.StringVar(&registry, "registry", defaultRegistry, "The micro service registry")
	flag.BoolVar(&test, "t", false, "Execute a go test")
	flag.BoolVar(&test, "test", false, "Execute a go test")

	flag.Usage = usage
	flag.Parse()

	args := flag.Args()

	if len(args) < 1 {
		usage()
	}

	fmt.Println("------------ gorun -------------")
	fmt.Printf("APOLLO_IP: %s\n", apolloIP)
	fmt.Printf("APOLLO_ENV: %s\n", cluster)
	fmt.Printf("APOLLO_APPID: %s\n", appID)

	if test {
		fmt.Println("--------------------------------")
		fmt.Printf("go test --run %s\n", args[0])
		fmt.Println("")
		cmd = exec.Command("go", "test", "--run", args[0])
	} else {
		fmt.Printf("registry=%s\n", registry)
		fmt.Println("--------------------------------")
		fmt.Printf("go run %s\n", args[0])
		fmt.Println("")
		cmd = exec.Command("go", "run", args[0])
	}

	if cmd == nil {
		return
	}

	var env = os.Environ()
	env = append(env, assembleEnv("APOLLO_IP", apolloIP))
	env = append(env, assembleEnv("APOLLO_ENV", cluster))
	env = append(env, assembleEnv("APOLLO_APPID", appID))
	env = append(env, assembleEnv("APOLLO_ACCESSKEY", key))
	env = append(env, assembleEnv("registry", registry))
	cmd.Env = env

	var errStdout, errStderr error
	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()

	go func() {
		_, errStdout = copyAndCapture(stdoutIn)
	}()
	go func() {
		_, errStderr = copyAndCapture(stderrIn)
	}()

	if err := cmd.Start(); err != nil {
		panic(err.Error())
	}

	err := cmd.Wait()
	if err != nil {
		log.Fatalf("gorun failed with %s\n", err)
	}
	if errStdout != nil || errStderr != nil {
		log.Fatalf("failed to capture stdout or stderr\n")
	}
}

func assembleEnv(key, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}

func copyAndCapture(r io.Reader) ([]byte, error) {
	var out []byte
	buf := make([]byte, 1024, 1024)
	for {
		n, err := r.Read(buf[:])
		if n > 0 {
			d := buf[:n]
			out = append(out, d...)
			os.Stdout.Write(d)
		}
		if err != nil {
			// Read returns io.EOF at the end of file, which is not an error for us
			if err == io.EOF {
				err = nil
			}
			return out, err
		}
	}
}
