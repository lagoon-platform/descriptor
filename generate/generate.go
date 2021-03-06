package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"text/template"
)

const (
	jsonFileName                   = "generate/engine_api.json"
	implementationTemplateFileName = "generate/generate_implementation.tmpl"
	interfaceTemplateFileName      = "generate/generate_interface.tmpl"
	generatedImplementations       = "generated%s.go"
	generatedInterfaces            = "tInterfaceGenerated.go"
)

//Interface represents and interface to generate
type Interface struct {
	//Name contains the name of the generated interface
	Name string `json:"name"`
	//Doc contains the documentation of the generated interface
	Doc string
	//Implentations holds the list of supported implementation to generate
	Implentations []string `json:"implemented_by"`
	//Methods holds the list of methods exposed by the interface
	Methods []Method `json:"methods"`
}

//Method represents method exposed by an interface
type Method struct {
	//Name represents the name of the method
	Name string
	//Doc represents the documentation of the method
	Doc string
	//Ret represents the type(s) returned by the method
	Ret string `json:"returns"`
	//Att represents the holder's attribute on which the method implementation is based
	Att string `json:"attribute"`
	//NoPointer should be set to 1 in order to dereference an pointer on the Att
	NoPointer int `json:"no_pointer"`
	//Custom holds the custom implementation of the method
	Custom customImplementation `json:"custom"`
	//ReturnTComponent defines a method which will return a TComponent interface
	ReturnTComponent returnTComponent `json:"component"`
	//ReturnTResolve defines a method which will return a generated interface by resolving a reference
	ReturnTResolve returnTResolve `json:"resolve"`
	//TInterface defines a method which will return a generated interface
	TInterface tInterface `json:"interface"`
	//TInterfaceMap defines a method which will return a map of a generated interface
	TInterfaceMap tInterfaceMap `json:"interface_map"`
	//TInterfaceArray defines a method which will return an array   a generated interface
	TInterfaceArray tInterfaceArray `json:"interface_array"`
}

type returnTComponent struct {
	Att  string `json:"attribute"`
	Type string `json:"type"`
}

type tInterface struct {
	Name string `json:"name"`
	returnTComponent
}

type returnTResolve struct {
	tInterface
}

type tInterfaceMap struct {
	tInterface
}

type tInterfaceArray struct {
	tInterface
}

type customImplementation struct {
	Impl string `json:"impl"`
	Ret  string `json:"returns"`
}

func getCreateInterface(noPointer bool, name, att, forType, fixReceiver string) string {
	r := "Create" + name + "For"
	if forType != "" {
		r += forType
	} else {
		r += att
	}
	if noPointer {
		if fixReceiver == "" {
			return r + "(*r.h." + att + ")"
		}
		return r + "(*" + fixReceiver + ")"
	} else if fixReceiver == "" {
		return r + "(r.h." + att + ")"
	} else {
		return r + "(" + fixReceiver + ")"
	}
}

func getHolderCall(indentation int, receiver, attribute, toCall string) string {
	var r string
	for i := 0; i < indentation; i++ {
		r += " "
	}
	r += receiver
	r += " := r.h."
	if attribute != "" {
		r += attribute
		r += "."
	}
	r += toCall
	r += "\n"
	return r
}

//Body returns the method body
func (m Method) Body() string {

	if m.TInterface.Name != "" {
		return "    return " + getCreateInterface(m.NoPointer == 1, m.TInterface.Name, m.TInterface.Att, m.TInterface.Type, "")
	} else if m.TInterfaceArray.Name != "" {
		r := "    result := make([]" + m.TInterfaceArray.Name + ", 0, 0)\n"
		r += "    for _ , val := range r.h." + m.TInterfaceArray.Att + "{\n"
		r += "        result = append(result, " + getCreateInterface(m.NoPointer == 1, m.TInterfaceArray.Name, m.TInterfaceArray.Att, m.TInterfaceArray.Type, "val") + ")\n"
		r += "    }\n"
		r += "    return result\n"
		return r

	} else if m.TInterfaceMap.Name != "" {
		r := "    result := make(map[string]" + m.TInterfaceMap.Name + ")\n"
		r += "    for k , val := range r.h." + m.TInterfaceMap.Att + "{\n"
		r += "        result[k] = " + getCreateInterface(m.NoPointer == 1, m.TInterfaceMap.Name, m.TInterfaceMap.Att, m.TInterfaceMap.Type, "val") + "\n"
		r += "    }\n"
		r += "    return result\n"
		return r
	} else if m.ReturnTResolve.Name != "" {
		r := getHolderCall(4, "v, err", m.ReturnTResolve.Att, "Resolve()")
		r += "    return " + getCreateInterface(false, m.ReturnTResolve.Name, m.ReturnTResolve.Att, m.ReturnTResolve.Type, "v")
		r += ", err"
		return r
	} else if m.ReturnTComponent.Att != "" || m.ReturnTComponent.Type != "" {
		r := getHolderCall(4, "v, err", m.ReturnTComponent.Att, "Component()")
		r += "    return " + getCreateInterface(false, "TComponent", m.ReturnTComponent.Att, m.ReturnTComponent.Type, "v")
		r += ", err"
		return r
	}
	if m.Custom.Impl != "" {
		return "return " + m.Custom.Impl
	}
	r := "return r.h." + m.Att
	return r
}

//ImplSignature returns the signature of the method for its implementation
func (m Method) ImplSignature(holder string) string {
	r := "func (r " + holder + ") "
	r += m.Signature()
	return r
}

//Signature returns the simple signature of the method
func (m Method) Signature() string {
	if m.TInterface.Name != "" {
		m.Ret = m.TInterface.Name
	} else if m.TInterfaceArray.Name != "" {
		m.Ret = "[]" + m.TInterfaceArray.Name
	} else if m.TInterfaceMap.Name != "" {
		m.Ret = "map[string]" + m.TInterfaceMap.Name
	} else if m.ReturnTResolve.Name != "" {
		m.Ret = "(" + m.ReturnTResolve.Name + ", error)"
	} else if m.ReturnTComponent.Att != "" || m.ReturnTComponent.Type != "" {
		m.Ret = "(TComponent, error)"
	} else if m.Custom.Ret != "" {
		m.Ret = m.Custom.Ret
	} else if m.Ret == "" {
		m.Ret = "string"
	}
	r := m.Name
	r += "() "
	r += m.Ret
	return r
}

//HolderSignature returns the signature of an older
func (i Interface) HolderSignature(s string) string {
	return i.Name + "On" + s + "Holder"
}

func main() {

	r, err := os.Open(jsonFileName)
	if err != nil {
		panic(err)
	}
	defer r.Close()

	v := []Interface{}
	err = json.NewDecoder(r).Decode(&v)
	if err != nil {
		panic(err)
	}

	//Generation of the interfaces implementation
	for _, i := range v {
		w, err := os.Create(fmt.Sprintf(generatedImplementations, i.Name))
		if err != nil {
			panic(err)
		}
		defer w.Close()

		t, err := template.ParseFiles(implementationTemplateFileName)
		if err != nil {
			panic(err)
		}

		var buf bytes.Buffer
		err = t.Execute(&buf, i)
		if err != nil {
			panic(err)
		}

		src, err := format.Source(buf.Bytes())
		if err != nil {
			panic(err)
		}

		_, err = w.Write(src)
		if err != nil {
			panic(err)
		}
	}
	//Generation of the interfaces definitions
	w, err := os.Create(generatedInterfaces)
	if err != nil {
		panic(err)
	}
	defer w.Close()

	t, err := template.ParseFiles(interfaceTemplateFileName)
	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, v)
	if err != nil {
		panic(err)
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		panic(err)
	}

	_, err = w.Write(src)
	if err != nil {
		panic(err)
	}
}
