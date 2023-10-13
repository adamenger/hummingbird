package parser

import ( 	
  "os"
  "fmt"
  "strings"
  "path/filepath"
  "github.com/trivago/grok"
  "bufio"
)

// GrokParser represents a parser that utilizes Grok patterns to extract
// structured data from unstructured log lines.
type GrokParser struct {
	Grok *grok.Grok
	Patterns map[string]string
	PatternsPath string
}

// NewGrokParser creates a new Grok parser.
func NewGrokParser(patternsPath string) (*GrokParser, error) {

  gp := &GrokParser{
    Patterns: make(map[string]string),
    PatternsPath: patternsPath,
  }
	gp.LoadPatterns()
	
  g, err := grok.New(grok.Config{
    Patterns: gp.Patterns,
  })
	if err != nil {
		return nil, err
	}
  gp.Grok = g

	return gp, nil
}

func (gp *GrokParser) LoadPatterns() error {
	// Check if the path points to a directory. If so, append the glob pattern to match .grok files
	if fi, err := os.Stat(gp.PatternsPath); err == nil {
		if fi.IsDir() {
			gp.PatternsPath = filepath.Join(gp.PatternsPath, "*.grok")
		}
	} else {
		return fmt.Errorf("invalid path: %s", gp.PatternsPath)
	}

	files, err := filepath.Glob(gp.PatternsPath)
	if err != nil {
		return err
	}

	for _, fileName := range files {
		file, err := os.Open(fileName)
		if err != nil {
			return err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) == 0 || line[0] == '#' {
				// Skip empty lines or lines starting with a hash (comments)
				continue
			}

			parts := strings.SplitN(line, " ", 2)
			if len(parts) != 2 {
				// Skip lines that don't seem to fit the expected pattern
				continue
			}

			key := parts[0]
			pattern := parts[1]
			gp.Patterns[key] = pattern
		}

		if err := scanner.Err(); err != nil {
			return err
		}
	}
	return nil

}
