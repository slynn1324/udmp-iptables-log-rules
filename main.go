package main

import (
    "fmt"
    "regexp"
    "os/exec"
    "strings"
    "bytes"
    "os"
    "flag"
)

// these values are replaced by build parms in build.sh
var versionNumber string = "0.0.0"
var commitId string = "none"

// a list of the iptables table names that we care about, and abbreviations for the log-prefix
var tableNames = map[string]string{
    "UBIOS_WAN_IN_USER":"WI",
    "UBIOS_WAN_OUT_USER":"W0",
    "UBIOS_WAN_LOCAL_USER":"WL",
    "UBIOS_LAN_IN_USER":"LI",
    "UBIOS_LAN_OUT_USER":"LO",
    "UBIOS_LAN_LOCAL_USER":"LL",
    "UBIOS_GUEST_IN_USER":"GI",
    "UBIOS_GUEST_OUT_USER":"GO",
    "UBIOS_GUEST_LOCAL_USER":"GL",
}

// regular expressions for patterns we'll be looking for
var tablePattern = regexp.MustCompile("-A (" + getTableNamesPattern() + ")")
var commentPattern = regexp.MustCompile("-m comment --comment (\\d+)")
var dropPattern = regexp.MustCompile("-j DROP")
var logPattern = regexp.MustCompile("--log-prefix \"\\[DROP [A-Z]{2} (\\d+)\\] \"")


/**
 * create a regex pattern by 'OR'-ing all of the keys from the tableNames map
 */
func getTableNamesPattern() string {
    rval := ""
    idx := 0
    for k, _ := range tableNames {
        if ( idx > 0 ){
            rval += "|"
        }
        rval += k
        idx++
    }
    
    return rval
}

/*
 * truncate a string by keeping the right-most <n> characters
 */
func leftTruncate(val string, n int) string {
    if ( len(val) > n ){
        return val[len(val)-n:]
    } else {
        return val
    }
}

/*
 * exec iptables-save to get a dump of the current rules
 * if the current user does not have permissions here, this will exit(1) the app
 */
func iptablesSave() []string {
    // exec iptables-save and capture the output
    // cmd := exec.Command("cat", "iptables-save.txt")
    cmd := exec.Command("iptables-save", "-c")
    cmdOut, err := cmd.Output()

    if err != nil {
        fmt.Println("error invoking iptables-save (did you run as root?)")
        fmt.Println(err.Error())
        os.Exit(1)
    }

    // split the output into lines
    lines := strings.Split(string(cmdOut), "\n")

    fmt.Printf("iptables-save returned %d lines\n", len(lines))

    return lines
}

/*
 * exec iptables-restore to load the updated rules
 * if the current user does not have permissions here, this will exit(2) the app
 */
func iptablesRestore(lines []string) {
    // turn the lines slice into a \n separated string
    output := strings.Join(lines, "\n")

    // pipe the output to iptables-restore
    buffer := bytes.Buffer{}
    buffer.Write([]byte(output))

    cmd := exec.Command("iptables-restore", "-c")
    //cmd = exec.Command("tee", "out.txt")
    cmd.Stdin = &buffer
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    err := cmd.Run() // run waits for completion
    if err != nil {
        fmt.Println("error invoking iptables-restore")
        fmt.Println(err)
        os.Exit(2)
    }
    fmt.Println("iptables-restore success")
}

/*
 * create new iptables log rules for all matching drop rules
 */
func create(){

    fmt.Println("creating iptables log rules")

    lines := iptablesSave()
    outLines := make([]string, 0)

    // keep track if the last line was a log statement to avoid writing duplicates
    isLog := false

    addCount := 0


    // process the lines, injecting log rules as needed
    for _, line := range lines {
        tableResult := tablePattern.FindStringSubmatch(line)
        if ( tableResult != nil ){

            commentResult := commentPattern.FindStringSubmatch(line)
            dropResult := dropPattern.FindStringSubmatch(line)

            // if this line has a matching comment and is a DROP statement
            if ( commentResult != nil && dropResult != nil ){

                // if the previous line was not a log statememnt already, add one
                if ( !isLog ){
                    tableName := tableResult[1]
                    ruleId := commentResult[1]

                    // need to truncate the rule id so it fits in the rule - take the last 14 characters.  iptables log-prefix is limited to 29 bytes
                    truncatedRuleId := leftTruncate(ruleId, 14)

                    cleanLine := commentPattern.ReplaceAllString(line, "")
                    cleanLine = dropPattern.ReplaceAllString(cleanLine, "")
                    
		            logLine := fmt.Sprintf("%s -m limit --limit 6/min -j LOG --log-prefix \"[DROP %s %s] \"", cleanLine, tableNames[tableName], truncatedRuleId)
                    outLines = append(outLines, logLine)

                    //fmt.Printf("Adding LOG for %s\n", line)
                    fmt.Printf("ADD RULE: %s\n", logLine)
                    addCount++
                } 

                // this was a drop line, so it's not a log line
                isLog = false

            } else {
                // check if this is a log line and set the state variable for the next iteration
                logResult := logPattern.FindStringSubmatch(line)
                if ( logResult != nil ){
                    isLog = true
                } else {
                    isLog = false
                }
            }
        } 

        // add the original line to the output
        outLines = append(outLines, line)
    }

    // if we added any log lines, then update the iptables rules
    if ( addCount > 0 ){
        iptablesRestore(outLines)
        fmt.Printf("%d iptables log rules added\n", addCount)
    } else {
        fmt.Println("no new log rules, not updating iptables")
    }

}

/**
 * delete previously created log rules
 *   - log rules detected based on logPattern at top
 */
func delete() {
    fmt.Println("deleting iptables drop log rules")

    lines := iptablesSave()

    outLines := make([]string, 0)

    count := 0

    for _, line := range lines {

        logMatch := logPattern.FindStringSubmatch(line)
        if ( logMatch != nil ){
            count++
            fmt.Printf("DELETE RULE: %s\n", line)
        } else {
            outLines = append(outLines, line)
        }
    }

    iptablesRestore(outLines)

    fmt.Printf("%d iptables log rules deleted\n", count)

}

/*
 * print version info
 */
func version() {
    fmt.Printf("v%s\ncommit=%s\n", versionNumber, commitId)
}

func main() {

    // process flags and run the right function

    versionMode := flag.Bool("v", false, "print version info")
    deleteMode := flag.Bool("d", false, "delete - cleans up previously created rules")

    flag.Parse()

    if ( *versionMode ){
        version()
    } else if ( *deleteMode ){
        delete()
    } else {
        create()
    }
}
