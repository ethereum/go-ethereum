package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Object struct {
	Name   string
	Groups []*Group
}

type Group struct {
	Vertexes []float32
	Normals  []float32
	Material *Material
}

type Material struct {
	Name      string
	Ambient   []float32
	Diffuse   []float32
	Specular  []float32
	Shininess float32
}

func Read(filename string) (map[string]*Object, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var materials map[string]*Material
	var objects = make(map[string]*Object)
	var object *Object
	var group *Group
	var vertex []float32
	var normal []float32

	lno := 0
	line := ""
	scanner := bufio.NewScanner(file)

	fail := func(msg string) error {
		return fmt.Errorf(msg+" at %s:%d: %s", filename, lno, line)
	}

	for scanner.Scan() {
		lno++
		line = scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		if fields[0] == "mtllib" {
			if len(fields) != 2 {
				return nil, fail("unsupported materials library line")
			}
			materials, err = readMaterials(filepath.Join(filepath.Dir(filename), fields[1]))
			if err != nil {
				return nil, err
			}
			continue
		}

		if fields[0] == "o" {
			if len(fields) != 2 {
				return nil, fail("unsupported object line")
			}
			object = &Object{Name: fields[1]}
			objects[object.Name] = object
			group = nil
			continue
		}

		if object == nil {
			return nil, fail("found data before object")
		}

		if fields[0] == "usemtl" {
			group = &Group{}
			object.Groups = append(object.Groups, group)
		}

		switch fields[0] {
		case "usemtl":
			if len(fields) != 2 {
				return nil, fail("unsupported material usage line")
			}
			group.Material = materials[fields[1]]
			if group.Material == nil {
				return nil, fmt.Errorf("material %q not defined", fields[1])
			}
		case "v":
			if len(fields) != 4 {
				return nil, fail("unsupported vertex line")
			}
			for i := 0; i < 3; i++ {
				f, err := strconv.ParseFloat(fields[i+1], 32)
				if err != nil {
					return nil, fail("cannot parse float")
				}
				vertex = append(vertex, float32(f))
			}
		case "vn":
			if len(fields) != 4 {
				return nil, fail("unsupported vertex normal line")
			}
			for i := 0; i < 3; i++ {
				f, err := strconv.ParseFloat(fields[i+1], 32)
				if err != nil {
					return nil, fail("cannot parse float")
				}
				normal = append(normal, float32(f))
			}
		case "f":
			if len(fields) != 4 {
				return nil, fail("unsupported face line")
			}
			for i := 0; i < 3; i++ {
				face := strings.Split(fields[i+1], "/")
				if len(face) != 3 {
					return nil, fail("unsupported face shape (not a triangle)")
				}
				vi, err := strconv.Atoi(face[0])
				if err != nil {
					return nil, fail("unsupported face vertex index")
				}
				ni, err := strconv.Atoi(face[2])
				if err != nil {
					return nil, fail("unsupported face normal index")
				}
				vi = (vi - 1) * 3
				ni = (ni - 1) * 3
				group.Vertexes = append(group.Vertexes, vertex[vi], vertex[vi+1], vertex[vi+2])
				group.Normals = append(group.Normals, normal[ni], normal[ni+1], normal[ni+2])
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return objects, nil
}

func readMaterials(filename string) (map[string]*Material, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("cannot read referenced material library: %v", err)
	}
	defer file.Close()

	var materials = make(map[string]*Material)
	var material *Material

	lno := 0
	line := ""
	scanner := bufio.NewScanner(file)

	fail := func(msg string) error {
		return fmt.Errorf(msg+" at %s:%d: %s", filename, lno, line)
	}

	for scanner.Scan() {
		lno++
		line = scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		if fields[0] == "newmtl" {
			if len(fields) != 2 {
				return nil, fail("unsupported material definition")
			}
			material = &Material{Name: fields[1]}
			material.Ambient = []float32{0.2, 0.2, 0.2, 1.0}
			material.Diffuse = []float32{0.8, 0.8, 0.8, 1.0}
			material.Specular = []float32{0.0, 0.0, 0.0, 1.0}
			materials[material.Name] = material
			continue
		}

		if material == nil {
			return nil, fail("found data before material")
		}

		switch fields[0] {
		case "Ka":
			if len(fields) != 4 {
				return nil, fail("unsupported ambient color line")
			}
			for i := 0; i < 3; i++ {
				f, err := strconv.ParseFloat(fields[i+1], 32)
				if err != nil {
					return nil, fail("cannot parse float")
				}
				material.Ambient[i] = float32(f)
			}
		case "Kd":
			if len(fields) != 4 {
				return nil, fail("unsupported diffuse color line")
			}
			for i := 0; i < 3; i++ {
				f, err := strconv.ParseFloat(fields[i+1], 32)
				if err != nil {
					return nil, fail("cannot parse float")
				}
				material.Diffuse[i] = float32(f)
			}
		case "Ks":
			if len(fields) != 4 {
				return nil, fail("unsupported specular color line")
			}
			for i := 0; i < 3; i++ {
				f, err := strconv.ParseFloat(fields[i+1], 32)
				if err != nil {
					return nil, fail("cannot parse float")
				}
				material.Specular[i] = float32(f)
			}
		case "Ns":
			if len(fields) != 2 {
				return nil, fail("unsupported shininess line")
			}
			f, err := strconv.ParseFloat(fields[1], 32)
			if err != nil {
				return nil, fail("cannot parse float")
			}
			material.Shininess = float32(f / 1000 * 128)
		case "d":
			if len(fields) != 2 {
				return nil, fail("unsupported transparency line")
			}
			f, err := strconv.ParseFloat(fields[1], 32)
			if err != nil {
				return nil, fail("cannot parse float")
			}
			material.Ambient[3] = float32(f)
			material.Diffuse[3] = float32(f)
			material.Specular[3] = float32(f)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Exporting from blender seems to show everything too dark in
	// practice, so hack colors to look closer to what we see there.
	// TODO This needs more real world checking.
	for _, material := range materials {
		if material.Ambient[0] == 0 && material.Ambient[1] == 0 && material.Ambient[2] == 0 && material.Ambient[3] == 1 {
			material.Ambient[0] = material.Diffuse[0] * 0.7
			material.Ambient[1] = material.Diffuse[1] * 0.7
			material.Ambient[2] = material.Diffuse[2] * 0.7
		}
		for i := 0; i < 3; i++ {
			material.Diffuse[i] *= 1.3
			if material.Diffuse[i] > 1 {
				material.Diffuse[i]  = 1
			}
		}
	}

	return materials, nil
}
