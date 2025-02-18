package rtec

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/PRETgroup/easy-rte/rtedef"
)

//Converter is the struct we use to store all functions for conversion (and what we operate from)
type Converter struct {
	Funcs     []rtedef.EnforcedFunction
	Language  string
	templates *template.Template
}

//New returns a new instance of a Converter based on the provided language
func New(language string, serialComposition bool, parallelComposition bool, synthesis bool) (*Converter, error) {
	switch strings.ToLower(language) {
	case "c":
		return &Converter{Funcs: make([]rtedef.EnforcedFunction, 0), Language: "c", templates: cTemplates}, nil
	case "vhdl":
		fmt.Println("WARNING: VHDL compilation support is currently not working due to problems with the VHDL type system. Try Verilog instead.")
		return &Converter{Funcs: make([]rtedef.EnforcedFunction, 0), Language: "vhdl", templates: vhdlTemplates}, nil
	case "verilog":
		if parallelComposition {
			return &Converter{Funcs: make([]rtedef.EnforcedFunction, 0), Language: "verilog", templates: verilogParallelCompositionTemplates}, nil
		} else if serialComposition {
			return &Converter{Funcs: make([]rtedef.EnforcedFunction, 0), Language: "verilog", templates: verilogSerialCompositionTemplates}, nil
		} else if synthesis {
			return &Converter{Funcs: make([]rtedef.EnforcedFunction, 0), Language: "verilog", templates: verilogSynthesisTemplates}, nil
		} else {
			return &Converter{Funcs: make([]rtedef.EnforcedFunction, 0), Language: "verilog", templates: verilogTemplates}, nil
		}
	default:
		return nil, errors.New("Language " + language + " is not supported")
	}
}

//AddFunction should be called for each Function in the project
func (c *Converter) AddFunction(functionbytes []byte) error {
	FB := rtedef.EnforcedFunction{}
	if err := xml.Unmarshal(functionbytes, &FB); err != nil {
		return errors.New("Couldn't unmarshal EnforcedFunction xml: " + err.Error())
	}

	c.Funcs = append(c.Funcs, FB)

	return nil
}

//OutputFile is used when returning the converted data from the iec61499
type OutputFile struct {
	Name      string
	Extension string
	Contents  []byte
}

//TemplateData is the structure used to hold data being passed into the templating engine
type TemplateData struct {
	FunctionIndex int
	Functions     []rtedef.EnforcedFunction
}

//ConvertAll converts iec61499 xml (stored as []FB) into vhdl []byte for each block (becomes []VHDLOutput struct)
//Returns nil error on success
func (c *Converter) ConvertAll(serialComposition bool, parallelComposition bool, synthesis bool) ([]OutputFile, error) {
	finishedConversions := make([]OutputFile, 0, len(c.Funcs))

	type templateInfo struct {
		Prefix    string
		Name      string
		Extension string
	}

	var templates []templateInfo

	//convert all functions
	if c.Language == "c" {
		templates = []templateInfo{
			{"F_", "functionC", "c"},
			{"F_", "functionH", "h"},
			{"cbmc_main_", "mainCBMCC", "c"},
		}
	}
	if c.Language == "vhdl" {
		templates = []templateInfo{
			{"F_", "functionVhdl", "vhdl"},
		}
	}
	if c.Language == "verilog" {
		if parallelComposition {
			fmt.Println("Note: Experimentally compiled as parallel composition for hardware synthesis")
			templates = []templateInfo{
				{"parallel_F_", "functionVerilog", "sv"},
			}
		} else if serialComposition {
			fmt.Println("Note: Experimentally compiled as serial composition for hardware synthesis")
			templates = []templateInfo{
				{"series_F_", "functionVerilog", "sv"},
			}
		} else if synthesis {
			fmt.Println("Note: Experimentally compiled for hardware synthesis")
			templates = []templateInfo{
				{"synthesis_F_", "functionVerilog", "sv"},
			}
		} else {
			templates = []templateInfo{
				{"test_F_", "functionVerilog", "sv"},
			}
		}
	}
	for _, template := range templates {
		for i := 0; i < len(c.Funcs); i++ {

			output := &bytes.Buffer{}
			fmt.Println("Right before where something i think long running lives")
			if err := c.templates.ExecuteTemplate(output, template.Name, TemplateData{FunctionIndex: i, Functions: c.Funcs}); err != nil {
				return nil, errors.New("Couldn't format template (fb) of" + c.Funcs[i].Name + ": " + err.Error())
			}
			fmt.Println("Right after it")

			finishedConversions = append(finishedConversions, OutputFile{Name: template.Prefix + c.Funcs[i].Name, Extension: template.Extension, Contents: output.Bytes()})
		}
	}

	return finishedConversions, nil
}
