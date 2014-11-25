package main

import (
    "bufio"
    "fmt"
    "log"
    "os"
    "regexp"
    "strings"
    "io/ioutil"
    "path/filepath"
)

func readFile(path string) (string, error) {
    data, err := ioutil.ReadFile(path)
    return string(data), err
}

func writeFile(path string, content string) (error) {
    data := []byte(content)
    return ioutil.WriteFile(path, data, 0x777)
}

func readLines(path string) (string, error) {
    file, err := os.Open(path)
    if err != nil {
        return "", err
    }
    defer file.Close()
    scanner := bufio.NewScanner(file)
    var lines []string
    for scanner.Scan() {
        lines = append(lines, scanner.Text())
    }
    return strings.Join(lines, "\n"), scanner.Err()
}

func handleVariables(text string) (string) {
    var re = regexp.MustCompile(`\{\s*(\$.+)\s*\}`)
    var_matches := re.FindAllStringSubmatch(text, -1)
    var output = ""

    for _, matches := range var_matches {
        output = ""
        decorator_split := strings.Split(matches[1], "|")
        variable := decorator_split[0]
        decorators := decorator_split[1:]
        index_split := strings.Split(variable, ".")
        variable_name := index_split[0]
        indexes := index_split[1:]

        output = variable_name

        for _, index := range indexes {
            if strings.Index(index, "$") == -1 {
                output = fmt.Sprintf("%s[\"%s\"]", output, index)
            } else {
                output = fmt.Sprintf("%s[%s]", output, index)
            }
        }

        s := regexp.MustCompile(`\:(\"^\"+\")?`)

        for _, decorator := range decorators {
            param_split := s.Split(decorator, -1)
            if len(param_split) == 1 {
            output = fmt.Sprintf("%s(%s)", param_split[0], output)
            } else {
                for _, param := range param_split[1:] {
                    output = fmt.Sprintf("%s, %s", output, param)
                }
                output = fmt.Sprintf("%s(%s)", param_split[0], output)
            }
        }
        text = strings.Replace(text, matches[0], fmt.Sprintf("<?= %s ?>", output), -1)
    }

    return text
}

func handleForeach(text string) (string) {
    var open_tag = regexp.MustCompile(`(/s+)?\{foreach\s+(.+)\}`)
    var close_tag = regexp.MustCompile(`\{\/foreach\}`)
    var else_tag = regexp.MustCompile(`\{foreachelse\}`)
    var from_value = regexp.MustCompile(`from\s*\=\s*(\$[a-zA-Z0-9_]+)`)
    var item_value = regexp.MustCompile(`item\s*\=\s*\"([a-zA-Z0-9_]+)\"`)

    /*
    lines := strings.Split(text, "\n")
    locs := make([]string, len(lines))

    for i, line := range lines  {
        if open_tag.MatchString(line) {
            locs[i] = "open"
        } else if close_tag.MatchString(line) {
            locs[i] = "close"
        } else if else_tag.MatchString(line) {
            locs[i] = "else"
        }
    }

    for j, val := range locs {
        if val == "else" {
           for k := j; k > 0; k-- {
                if locs[k] == "open" {
                   fmt.Println("if around lines %s and %s", k, j)
                }
            }
            for x := j; x < len(locs); x++ {
                if locs[x] == "close" {
                    fmt.Println("else around lines %s and %s", j, x)
                }
            }
        }
    }
    */

    loops := open_tag.FindAllStringSubmatch(text, -1)

    for _, matches := range loops {
        parameters := matches[1]
        item := item_value.FindStringSubmatch(parameters)
        from := from_value.FindStringSubmatch(parameters)
        php_loop := fmt.Sprintf ("<? foreach (%s as $%s) { ?>", from[1], item[1])
        text = strings.Replace(text, matches[0], php_loop, -1)
    }

    text = close_tag.ReplaceAllString(text, "<? } ?>")
    text = else_tag.ReplaceAllString(text, "<? /* foreach else goes here */ ?>")
    return text
}

func handleComments(text string) (string) {
    var comment_open = regexp.MustCompile(`\{\*`)
    var comment_close = regexp.MustCompile(`\*\}`)
    text = comment_open.ReplaceAllString(text, "<? /* ")
    text = comment_close.ReplaceAllString(text, " */ ?>")
    return text
}

func dirWalk(path string, f os.FileInfo, err error) error {
    if !f.IsDir() && (filepath.Ext(path) == ".tpl") {
        go convertTemplate(path)
    }
    return nil
}

func convertTemplate(path string) error {
    fmt.Sprintf("template: %s", path)
    text, _ := readFile(path)
    text = handleComments(text)
    text = handleVariables(text)
    text = handleForeach(text)
    //fmt.Print(text)
    return writeFile(fmt.Sprintf("%s.php", path), text)
}


func main() {
    for i, arg := range os.Args {
        fmt.Println(i, arg)
    }

    //text, err := readLines("/home/phil/Desktop/templates/test.tpl")
    source_dir := os.Args[1]
    if source_dir == "" {
        log.Fatalf("No path provided.")
    }

    filepath.Walk(source_dir, dirWalk)

    /*

    */
}
