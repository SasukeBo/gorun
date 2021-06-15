package main

import (
	"flag"
	"fmt"
	color "gopkg.in/gookit/color.v1"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	info    = color.Notice.Render
	warn    = color.Warn.Render
	success = color.Success.Render
	danger  = color.Danger.Render

	version = "v0.0.1"
)

var usageStr = fmt.Sprintf(`
Version: %s

Usage: gorun [options] <subject>

Example: gorun -id gorun main.go

Options:
	-ip, --apollo_ip  <url>                 APOLLO API server URL
	-c,  --cluster    <cluster name>        APOLLO cluster name
	-id, --app_id     <app id>              APOLLO app id
	-k,  --key        <access key>          APOLLO access key
	-r,  --registry   <service registry>    Micro service registry
	-t,  --test                             Go test
	--print                                 Print env
`, version)

var (
	defaultApolloIP = "apollo.api.thingyouwe.com"
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
		isPrint  bool
		cmd      *exec.Cmd
	)

	if v := os.Getenv("APOLLO_IP"); len(v) != 0 {
		defaultApolloIP = v
	}

	if v := os.Getenv("APOLLO_ENV"); len(v) != 0 {
		defaultCluster = v
	}

	if v := os.Getenv("registry"); len(v) != 0 {
		defaultRegistry = v
	}

	flag.StringVar(&apolloIP, "ip", defaultApolloIP, "The Apollo config api server URL")
	flag.StringVar(&apolloIP, "apollo_ip", defaultApolloIP, "The Apollo config api server URL")
	flag.StringVar(&cluster, "c", defaultCluster, "The Apollo cluster name")
	flag.StringVar(&cluster, "cluster", defaultCluster, "The Apollo cluster name")
	flag.StringVar(&appID, "id", "", "The Apollo app id")
	flag.StringVar(&appID, "app_id", "", "The Apollo app id")
	flag.StringVar(&key, "k", "default_key", "The Apollo access key")
	flag.StringVar(&key, "key", "default_key", "The Apollo access key")
	flag.StringVar(&registry, "r", defaultRegistry, "The micro service registry")
	flag.StringVar(&registry, "registry", defaultRegistry, "The micro service registry")
	flag.BoolVar(&test, "t", false, "Execute a go test")
	flag.BoolVar(&test, "test", false, "Execute a go test")
	flag.BoolVar(&isPrint, "print", false, "Print env")

	flag.Usage = usage
	flag.Parse()

	args := flag.Args()

	if len(args) < 1 && !isPrint {
		usage()
	}

	var env = os.Environ()
	if isEmpty(apolloIP) {
		fmt.Println(danger("apollo_ip cannot be empty."))
		return
	}
	env = append(env, assembleEnv("APOLLO_IP", apolloIP))
	if isEmpty(cluster) {
		fmt.Println(danger("cluster cannot be empty."))
		return
	}
	env = append(env, assembleEnv("APOLLO_ENV", cluster))
	if isEmpty(appID) {
		dir, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		appID = filepath.Base(dir)
	}
	env = append(env, assembleEnv("APOLLO_APPID", appID))
	if isEmpty(key) {
		fmt.Println(danger("key cannot be empty."))
		return
	}
	env = append(env, assembleEnv("APOLLO_ACCESSKEY", key))
	if isEmpty(registry) {
		fmt.Println(danger("registry cannot be empty."))
		return
	}
	env = append(env, assembleEnv("registry", registry))

	fmt.Println(info("------------ gorun -------------\n"))
	fmt.Printf("%s %s\n", warn("APOLLO_IP:"), apolloIP)
	fmt.Printf("%s %s\n", warn("APOLLO_ENV:"), cluster)
	fmt.Printf("%s %s\n", warn("APOLLO_APPID:"), appID)

	if test {
		fmt.Println(info("\n--------------------------------"))
		fmt.Printf("\n%s go test -v --run %s\n\n", info("[Test]"), args[0])
		cmd = exec.Command("go", "test", "-v", "--run", args[0])
	} else if isPrint {
		fmt.Printf("%s %s\n", warn("registry:"), registry)
		fmt.Println(info("\n--------------------------------"))
		fmt.Printf("\n%s service env\n\n", info("[Print]"))
		return
	} else {
		fmt.Printf("%s %s\n", warn("registry:"), registry)
		fmt.Println(info("\n--------------------------------"))
		fmt.Printf("\n%s service starting ...\n\n", info(fmt.Sprintf("[%s]", appID)))
		cmd = exec.Command("go", "run", args[0])
	}

	if cmd == nil {
		return
	}

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
		fmt.Print(danger(fmt.Sprintf("\ngorun failed with %s\n", err)))
		return
	}
	if errStdout != nil || errStderr != nil {
		fmt.Print(danger(fmt.Sprintf("\nfailed to capture stdout or stderr\n")))
		return
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
			_, _ = os.Stdout.Write(d)
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

func isEmpty(value string) bool {
	return len(value) == 0 || strings.HasPrefix(value, "-")
}
