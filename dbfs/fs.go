package main

import (
	"flag"
	"io/ioutil"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/golang/glog"
	"gopkg.in/v1/yaml"
)

type Query struct {
	Name        string
	Description string
	Cmd         string
	inumber     uint64
}

type QueryResult struct {
	Result []byte
}

type Update struct {
	Name        string
	Description string
	Cmd         string
	inumber     uint64
}

type UpdateStep struct {
}

type Spec struct {
	Queries []Query
	Updates []Update
}

func isWriteFlags(flags fuse.OpenFlags) bool {
	// TODO read/writeness are not flags, use O_ACCMODE
	return flags&fuse.OpenFlags(os.O_WRONLY|os.O_RDWR|os.O_APPEND|os.O_CREATE) != 0
}

func (q Query) Open(req *fuse.OpenRequest, res *fuse.OpenResponse, intr fs.Intr) (fs.Handle, fuse.Error) {
	glog.V(2).Infof("open on %#v", req)
	//	if isWriteFlags(req.Flags) {
	//		return nil, fuse.EPERM
	//	}
	return &QueryResult{[]byte(q.Cmd)}, nil
}

func (q Query) Attr() fuse.Attr {
	return fuse.Attr{Inode: q.inumber, Mode: 0444}
}

func (qr QueryResult) ReadAll(intr fs.Intr) ([]byte, fuse.Error) {
	return qr.Result, nil
}

func (qr QueryResult) Release(req *fuse.ReleaseRequest, intr fs.Intr) fuse.Error {
	glog.V(2).Infof("release on %#v", req)
	return nil
}

func (u Update) Open(req *fuse.OpenRequest, res *fuse.OpenResponse, intr fs.Intr) (fs.Handle, fuse.Error) {
	glog.V(2).Infof("open on %#v", req)
	//	if isWriteFlags(req.Flags) {
	//		return nil, fuse.EPERM
	//	}
	return &UpdateStep{}, nil
}

func (u Update) Attr() fuse.Attr {
	return fuse.Attr{Inode: u.inumber, Mode: 0222}
}

func (u UpdateStep) Write(req *fuse.WriteRequest, res *fuse.WriteResponse, intr fs.Intr) fuse.Error {
	glog.V(2).Infof("write on %#v: %v", req, string(req.Data))
	res.Size = len(req.Data)
	return nil
}

func (u UpdateStep) Flush(req *fuse.FlushRequest, intr fs.Intr) fuse.Error {
	glog.V(2).Infof("flush on %#v", req)
	return nil
}

func (u UpdateStep) Release(req *fuse.ReleaseRequest, intr fs.Intr) fuse.Error {
	glog.V(2).Infof("release on %#v", req)
	return nil
}

func (spec Spec) Root() (fs.Node, fuse.Error) {
	return spec, nil
}

func (spec Spec) Attr() fuse.Attr {
	return fuse.Attr{Inode: 1, Mode: os.ModeDir | 0555}
}

func (spec Spec) Lookup(name string, intr fs.Intr) (fs.Node, fuse.Error) {
	for _, q := range spec.Queries {
		if name == q.Name {
			return q, nil
		}
	}
	for _, u := range spec.Updates {
		if name == u.Name {
			return u, nil
		}
	}
	return nil, fuse.ENOENT
}

func (spec Spec) ReadDir(intr fs.Intr) ([]fuse.Dirent, fuse.Error) {
	dirDirs := make([]fuse.Dirent, 0)
	for _, q := range spec.Queries {
		dirDirs = append(dirDirs, fuse.Dirent{Inode: q.inumber, Name: q.Name, Type: fuse.DT_File})
	}
	for _, u := range spec.Updates {
		dirDirs = append(dirDirs, fuse.Dirent{Inode: u.inumber, Name: u.Name, Type: fuse.DT_File})
	}
	return dirDirs, nil
}

func readSpec(filename string) (Spec, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return Spec{}, err
	}
	var spec Spec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return Spec{}, err
	}

	inumber := uint64(1000)
	for _, q := range spec.Queries {
		q.inumber = inumber
		inumber++
	}
	for _, u := range spec.Updates {
		u.inumber = inumber
		inumber++
	}
	return spec, nil
}

var mountpoint = flag.String("mtpt", "", "directory to mount")
var specfile = flag.String("spec", "", "spec/json file")

func main() {
	flag.Parse()
	if *mountpoint == "" || *specfile == "" {
		glog.Error("check the args")
		os.Exit(2)
	}

	spec, err := readSpec(*specfile)
	if err != nil {
		glog.Error("failed to read spec from", *specfile, ":", err)
		os.Exit(2)
	}

	c, err := fuse.Mount(*mountpoint)
	if err != nil {
		glog.Fatal(err)
	}
	defer c.Close()

	err = fs.Serve(c, spec)
	if err != nil {
		glog.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		glog.Fatal(err)
	}
}
