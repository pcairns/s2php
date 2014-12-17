package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func readFile(path string) (string, error) {
	data, err := ioutil.ReadFile(path)
	return string(data), err
}

func writeFile(path string, content string) error {
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

func handleVariables(text string) string {
	var re = regexp.MustCompile(`\{\s*(\$[^\s\}]+)\s*\}`)
	var_matches := re.FindAllStringSubmatch(text, -1)
	var output = ""
	replacements := make(map[string]string)

	replacements["lower"] = "strtolower"
	replacements["escape"] = "htmlentities"

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
				if method_name, ok := replacements[param_split[0]]; ok {
					output = fmt.Sprintf("%s(%s)", method_name, output)
				} else {
					output = fmt.Sprintf("%s(%s)", param_split[0], output)
				}
			} else {
				for _, param := range param_split[1:] {
					output = fmt.Sprintf("%s, %s", output, param)
				}
				output = fmt.Sprintf("%s(%s)", param_split[1], output)
			}
		}
		text = strings.Replace(text, matches[0], fmt.Sprintf("<?= %s ?>", output), -1)
	}

	return text
}

func handleForeach(text string) string {
	var open_tag = regexp.MustCompile(`\{foreach\s+([^\}]+)\}`)
	var close_tag = regexp.MustCompile(`\{\/foreach\}`)
	var else_tag = regexp.MustCompile(`\{foreachelse\}`)
	var from_value = regexp.MustCompile(`from\s*\=\s*[\"|\']?\$([\S]+)[\"|\']?`)
	var item_value = regexp.MustCompile(`item\s*\=\s*[\"|\']?([a-zA-Z0-9_]+)[\"|\']?`)

	loops := open_tag.FindAllStringSubmatch(text, -1)

	for _, matches := range loops {
		parameters := matches[1]
		fmt.Print("CRAP:")
		fmt.Print(parameters)
		item := item_value.FindStringSubmatch(parameters)
		from := from_value.FindStringSubmatch(parameters)
		item_name := item[1]
		from_name := from[1]
		php_loop := fmt.Sprintf("<? foreach ($%s as $%s) { ?>", from_name, item_name)
		text = strings.Replace(text, matches[0], php_loop, -1)
	}

	text = close_tag.ReplaceAllString(text, "<? } ?>")
	text = else_tag.ReplaceAllString(text, "<? /* foreach else goes here */ ?>")
	return text
}

func handleComments(text string) string {
	var comment_open = regexp.MustCompile(`\{\*`)
	var comment_close = regexp.MustCompile(`\*\}`)
	text = comment_open.ReplaceAllString(text, "<? /* ")
	text = comment_close.ReplaceAllString(text, " */ ?>")
	return text
}

func handleIfStatements(text string) string {
	if_tag := regexp.MustCompile(`\{if\s+([^\}]+)\}`)

	if_statements := if_tag.FindAllStringSubmatch(text, -1)

	for _, matches := range if_statements {
		tag := matches[0]
		condition := matches[1]
		new_tag := fmt.Sprintf("<? if ( %s ) : ?>", condition)
		text = strings.Replace(text, tag, new_tag, -1)
	}

	text = strings.Replace(text, "{else}", "<? else : ?>", -1)
	text = strings.Replace(text, "{/if}", "<? endif; ?>", -1)
	return text
}

func handleMvcLinks(text string) string {
	tags := regexp.MustCompile(`\{mvc\_link\s+(.+)\}`)
	controller := regexp.MustCompile(`controller\=[\"|\']?([a-zA-Z0-9_$]+)[\"|\']?`)
	action := regexp.MustCompile(`action\=[\"|\']?([a-zA-Z0-9_$]+)[\"|\']?`)

	links := tags.FindAllStringSubmatch(text, -1)

	for _, matches := range links {
		tag := matches[0]
		controller_value := controller.FindAllStringSubmatch(tag, -1)
		action_value := action.FindAllStringSubmatch(tag, -1)
		new_tag := ""
		if len(controller_value) < 1 {
			continue // no controller skip these for the time being
		}

		if len(action_value) > 1 {
			new_tag = fmt.Sprintf("<? $h->url_for( \"%s/%s\" ) ?>", controller_value[0][1], action_value[0][1])
		} else {
			new_tag = fmt.Sprintf("<? $h->url_for( \"%s/index\" ) ?>", controller_value[0][1])
		}
		text = strings.Replace(text, tag, new_tag, -1)
	}

	return text
}

func stripLiteral(text string) string {
	tag := regexp.MustCompile(`\{\/?literal\}`)
	return tag.ReplaceAllString(text, "")
}

func handleIncludes(text string) string {
	pattern := regexp.MustCompile(`\{include\s+file=[\"|\']?([^\}\s\"\']+)[\"|\']?\}`)
	includes := pattern.FindAllStringSubmatch(text, -1)

	for _, matches := range includes {
		tag := matches[0]
		new_path := strings.Replace(matches[1], ".tpl", ".php", -1)
		new_tag := fmt.Sprintf("<? $this->partial( \"%s\" ) ?>", new_path)
		text = strings.Replace(text, tag, new_tag, -1)
	}

	return text
}

func handleScript(text string) string {

	pattern := regexp.MustCompile(`\{script\s*([\^\}]+)\}`)
	base_pattern := regexp.MustCompile(`base\=[\'|\"]?([^\s\"\']+)[\'|\"]?`)
	src_pattern := regexp.MustCompile(`src\=[\'|\"]?([^\s\"\']+)[\'|\"]?`)

	scripts := pattern.FindAllStringSubmatch(text, -1)

	for _, match := range scripts {
		script := match[0]
		base := base_pattern.FindStringSubmatch(script)
		src := src_pattern.FindStringSubmatch(script)
	}

	return text
}

func handleAssigns(text string) string {
	pattern := regexp.MustCompile(`\{assign\s+([^\}]+)\}`)
	name := regexp.MustCompile(`var\=[\'|\"]?([^\"\'\}\s]+)[\'|\"]?`)
	value := regexp.MustCompile(`value\=\s*([\'|\"]?[^\"\'\}\s]*[\'|\"]?)`)
	assigns := pattern.FindAllStringSubmatch(text, -1)
	for _, matches := range assigns {
		tag := matches[0]
		name_match := name.FindStringSubmatch(tag)
		value_match := value.FindStringSubmatch(tag)
		new_tag := fmt.Sprintf("<? $%s = %s; ?>", name_match[1], value_match[1])

		text = strings.Replace(text, tag, new_tag, -1)
	}

	return text
}

func handleArrayIndices(text string) string {
	pattern := regexp.MustCompile(`(\$[^\.\s\}]+)\.([^\.\s\}]+)`)
	matches := pattern.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		current := match[0]
		var replacement string
		if strings.Index(match[2], "$") == -1 {
			replacement = fmt.Sprintf("%s[\"%s\"]", match[1], match[2])
		} else {
			replacement = fmt.Sprintf("%s[%s]", match[1], match[2])
		}
		text = strings.Replace(text, current, replacement, -1)
	}

	return text
}

func handleFckeditor(text string) string {
	pattern := regexp.MustCompile(`\{fckeditor\s+([^\}]+)\}`)
	items := pattern.FindAllStringSubmatch(text, -1)

	for _, match := range items {
		tag := match[0]
		param_string := paramsToPhpArray(tag)
		new_tag := fmt.Sprintf("<? $h->fckeditor( %s ) ?>", param_string)

		text = strings.Replace(text, tag, new_tag, -1)
	}

	return text
}

func paramsToPhpArray(text string) string {
	key_value_pairs := regexp.MustCompile(`([^=\s]*)=[\"|\']([^"]*|[^=\s]*)[\"|\']`)

	params := key_value_pairs.FindAllStringSubmatch(text, -1)
	param_string := "array("

	for i, param := range params {
		if i > 0 {
			param_string += ", "
		}
		param_string += fmt.Sprintf("\"%s\" => \"%s\"", param[1], param[2])
	}

	param_string += ")"

	return param_string
}

func handleSubNavItem(text string) string {
	subnav_items := regexp.MustCompile(`\{subnav\_item\s+(.+)\}`)
	items := subnav_items.FindAllStringSubmatch(text, -1)

	for _, matches := range items {
		tag := matches[0]
		param_string := paramsToPhpArray(tag)
		new_tag := fmt.Sprintf("<? $h->subnavItem( %s ) ?>", param_string)
		text = strings.Replace(text, tag, new_tag, -1)
	}

	return text
}

func dirWalk(path string, f os.FileInfo, err error) error {
	if !f.IsDir() && (filepath.Ext(path) == ".tpl") {
		err := convertTemplate(path)
		if err != nil {
			fmt.Println(err)
		}

	}
	return nil
}

func convertTemplate(path string) error {
	fmt.Println(path)
	text, _ := readFile(path)
	text = handleArrayIndices(text)
	text = handleComments(text)
	text = handleVariables(text)
	text = handleForeach(text)
	text = handleIfStatements(text)
	text = handleIncludes(text)
	text = handleMvcLinks(text)
	text = handleSubNavItem(text)
	text = handleAssigns(text)
	text = handleFckeditor(text)
	text = stripLiteral(text)
	return writeFile(fmt.Sprintf("/home/phil/smarty/%s.php", filepath.Base(path)), text)
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
