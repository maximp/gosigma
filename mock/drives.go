// Copyright 2014 ALTOROS
// Licensed under the AGPLv3, see LICENSE file for details.

package mock

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/Altoros/gosigma/data"
)

type DriveLibrary struct {
	s sync.Mutex
	m map[string]*data.Drive
	p string
}

var Drives = &DriveLibrary{
	m: make(map[string]*data.Drive),
	p: "/api/2.0/drives",
}
var LibDrives = &DriveLibrary{
	m: make(map[string]*data.Drive),
	p: "/api/2.0/libdrives",
}

func ResetDrives() {
	Drives.Reset()
	LibDrives.Reset()
}

func InitDrive(d *data.Drive) (*data.Drive, error) {
	if d.UUID == "" {
		uuid, err := GenerateUUID()
		if err != nil {
			return nil, err
		}
		d.UUID = uuid
	}
	if d.Status == "" {
		d.Status = "unmounted"
	}

	return d, nil
}

func (d *DriveLibrary) Add(drv *data.Drive) error {
	drv, err := InitDrive(drv)
	if err != nil {
		return err
	}

	d.s.Lock()
	defer d.s.Unlock()

	d.m[drv.UUID] = drv

	return nil
}

func (d *DriveLibrary) AddDrives(dd []data.Drive) []string {
	d.s.Lock()
	defer d.s.Unlock()

	var result []string
	for _, drv := range dd {
		drv, err := InitDrive(&drv)
		if err != nil {
			d.m[drv.UUID] = drv
			result = append(result, drv.UUID)
		}
	}
	return result
}

func (d *DriveLibrary) Remove(uuid string) bool {
	d.s.Lock()
	defer d.s.Unlock()

	_, ok := d.m[uuid]
	delete(d.m, uuid)

	return ok
}

func (d *DriveLibrary) Reset() {
	d.s.Lock()
	defer d.s.Unlock()
	d.m = make(map[string]*data.Drive)
}

func (d *DriveLibrary) SetStatus(uuid, status string) {
	d.s.Lock()
	defer d.s.Unlock()

	drv, ok := d.m[uuid]
	if ok {
		drv.Status = status
	}
}

var ErrNotFound = errors.New("not found")

func (d *DriveLibrary) Clone(uuid string) (string, error) {
	d.s.Lock()
	defer d.s.Unlock()

	drv, ok := d.m[uuid]
	if !ok {
		return "", ErrNotFound
	}

	newUUID, err := GenerateUUID()
	if err != nil {
		return "", err
	}

	var newDrive data.Drive = *drv
	newDrive.Resource = *data.MakeDriveResource(newUUID)
	newDrive.Status = "unmounted"

	if d == LibDrives {
		Drives.Add(&newDrive)
	} else {
		d.m[newUUID] = &newDrive
	}

	return newUUID, nil
}

func (d *DriveLibrary) Resize(uuid string, size uint64) error {
	d.s.Lock()
	defer d.s.Unlock()

	drv, ok := d.m[uuid]
	if !ok {
		return ErrNotFound
	}

	drv.Size = size

	return nil
}

func (d *DriveLibrary) handleRequest(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")
	path = strings.TrimPrefix(path, d.p)
	path = strings.TrimPrefix(path, "/")

	switch r.Method {
	case "GET":
		d.handleGet(w, r, path)
	case "POST":
		d.handlePost(w, r, path)
	case "DELETE":
		d.handleDelete(w, r, path)
	}
}

func (d *DriveLibrary) handleGet(w http.ResponseWriter, r *http.Request, path string) {
	switch path {
	case "":
		d.handleDrives(w, r)
	case "detail":
		d.handleDrivesDetail(w, r, 200, nil)
	default:
		d.handleDrive(w, r, 200, path)
	}
}

func (d *DriveLibrary) handlePost(w http.ResponseWriter, r *http.Request, path string) {
	uuid := strings.TrimSuffix(path, "/action")
	d.handleAction(w, r, uuid)
}

func (d *DriveLibrary) handleDelete(w http.ResponseWriter, r *http.Request, uuid string) {
	if ok := d.Remove(uuid); !ok {
		h := w.Header()
		h.Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(404)
		w.Write([]byte(jsonNotFound))
		return
	}
	w.WriteHeader(204)
}

func (d *DriveLibrary) handleDrives(w http.ResponseWriter, r *http.Request) {
	d.s.Lock()
	defer d.s.Unlock()

	var dd data.Drives
	dd.Meta.TotalCount = len(d.m)
	dd.Objects = make([]data.Drive, 0, len(d.m))
	for _, drv := range d.m {
		var drv0 data.Drive
		drv0.Resource = drv.Resource
		drv0.Owner = drv.Owner
		drv0.Status = drv.Status
		dd.Objects = append(dd.Objects, drv0)
	}

	data, err := json.Marshal(&dd)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("500 " + err.Error()))
		return
	}

	h := w.Header()
	h.Set("Content-Type", "application/json; charset=utf-8")
	w.Write(data)
}

func (d *DriveLibrary) handleDrivesDetail(w http.ResponseWriter, r *http.Request, okcode int, filter []string) {
	d.s.Lock()
	defer d.s.Unlock()

	var dd data.Drives

	if len(filter) == 0 {
		dd.Meta.TotalCount = len(d.m)
		dd.Objects = make([]data.Drive, 0, len(d.m))
		for _, drv := range d.m {
			dd.Objects = append(dd.Objects, *drv)
		}
	} else {
		dd.Meta.TotalCount = len(filter)
		dd.Objects = make([]data.Drive, 0, len(filter))
		for _, uuid := range filter {
			if drv, ok := d.m[uuid]; ok {
				dd.Objects = append(dd.Objects, *drv)
			}
		}
	}

	data, err := json.Marshal(&dd)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("500 " + err.Error()))
		return
	}

	h := w.Header()
	h.Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(okcode)
	w.Write(data)
}

func (d *DriveLibrary) handleDrive(w http.ResponseWriter, r *http.Request, okcode int, uuid string) {
	d.s.Lock()
	defer d.s.Unlock()

	h := w.Header()

	drv, ok := d.m[uuid]
	if !ok {
		h.Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(404)
		w.Write([]byte(jsonNotFound))
		return
	}

	data, err := json.Marshal(&drv)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("500 " + err.Error()))
		return
	}

	h.Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(okcode)
	w.Write(data)
}

func (d *DriveLibrary) handleAction(w http.ResponseWriter, r *http.Request, uuid string) {
	vv := r.URL.Query()

	v, ok := vv["do"]
	if !ok || len(v) < 1 {
		w.WriteHeader(400)
		return
	}

	action := v[0]
	switch action {
	case "clone":
		d.handleClone(w, r, uuid)
	case "resize":
		d.handleResize(w, r, uuid)
	default:
		w.WriteHeader(400)
	}
}

func (d *DriveLibrary) handleClone(w http.ResponseWriter, r *http.Request, uuid string) {
	newUUID, err := d.Clone(uuid)
	if err == ErrNotFound {
		h := w.Header()
		h.Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(404)
		w.Write([]byte(jsonNotFound))
		return
	} else if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("500 " + err.Error()))
		return
	}
	Drives.handleDrivesDetail(w, r, 202, []string{newUUID})
}

func (d *DriveLibrary) handleResize(w http.ResponseWriter, r *http.Request, uuid string) {
	bb, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("500 " + err.Error()))
		return
	}

	drv, err := data.ReadDrive(bytes.NewReader(bb))
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("500 " + err.Error()))
		return
	}

	err = d.Resize(uuid, drv.Size)
	if err == ErrNotFound {
		h := w.Header()
		h.Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(404)
		w.Write([]byte(jsonNotFound))
		return
	} else if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("500 " + err.Error()))
		return
	}

	d.handleDrivesDetail(w, r, 202, []string{uuid})
}
