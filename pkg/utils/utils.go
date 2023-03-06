package utils

import (
	"fmt"
	"mime"
	"net"
	"os"
	"os/user"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-isatty"
)

// Min returns min of 2 int
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Exists checks if the file/folder in given path exists
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err) //skip mutate
}

// SplitDir splits a path with default path list separator or comma.
func SplitDir(d string) []string {
	dd := strings.Split(d, string(os.PathListSeparator))
	if len(dd) == 1 {
		dd = strings.Split(dd[0], ",")
	}
	return dd
}

// GetLocalIp get the local ip used to access remote address.
func GetLocalIp(address string) (string, error) {
	conn, err := net.Dial("udp", address)
	if err != nil {
		return "", err
	}
	ip, _, err := net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		return "", err
	}
	return ip, nil
}

func WithTimeout(f func() error, timeout time.Duration) error {
	var done = make(chan int, 1)
	var t = time.NewTimer(timeout)
	var err error
	go func() {
		err = f()
		done <- 1
	}()
	select {
	case <-done:
		t.Stop()
	case <-t.C:
		err = fmt.Errorf("timeout after %s", timeout)
	}
	return err
}

func RemovePassword(uri string) string {
	p := strings.Index(uri, "@")
	if p < 0 {
		return uri
	}
	sp := strings.Index(uri, "://") + 3
	if sp == 2 {
		sp = 0
	}
	cp := strings.Index(uri[sp:], ":")
	if cp < 0 || sp+cp > p {
		return uri
	}
	return uri[:sp+cp] + ":****" + uri[p:]
}

func GuessMimeType(key string) string {
	mimeType := mime.TypeByExtension(path.Ext(key))
	if !strings.ContainsRune(mimeType, '/') {
		mimeType = "application/octet-stream"
	}
	return mimeType
}

func StringContains(s []string, e string) bool {
	for _, item := range s {
		if item == e {
			return true
		}
	}
	return false
}

func FormatBytes(n uint64) string {
	if n < 1024 {
		return fmt.Sprintf("%d Bytes", n)
	}
	units := []string{"K", "M", "G", "T", "P", "E"}
	m := n
	i := 0
	for ; i < len(units)-1 && m >= 1<<20; i++ {
		m = m >> 10
	}
	return fmt.Sprintf("%.2f %siB (%d Bytes)", float64(m)/1024.0, units[i], n)
}

func SupportANSIColor(fd uintptr) bool {
	return isatty.IsTerminal(fd) && runtime.GOOS != "windows"
}

var uids = make(map[int]string)
var gids = make(map[int]string)
var users = make(map[string]int)
var groups = make(map[string]int)
var mutex sync.Mutex

var logger = GetLogger("kos")

func UserName(uid int) string {
	mutex.Lock()
	defer mutex.Unlock()
	name, ok := uids[uid]
	if !ok {
		if u, err := user.LookupId(strconv.Itoa(uid)); err == nil {
			name = u.Username
		} else {
			logger.Warnf("lookup uid %d: %s", uid, err)
			name = strconv.Itoa(uid)
		}
		uids[uid] = name
	}
	return name
}

func GroupName(gid int) string {
	mutex.Lock()
	defer mutex.Unlock()
	name, ok := gids[gid]
	if !ok {
		if g, err := user.LookupGroupId(strconv.Itoa(gid)); err == nil {
			name = g.Name
		} else {
			logger.Warnf("lookup gid %d: %s", gid, err)
			name = strconv.Itoa(gid)
		}
		gids[gid] = name
	}
	return name
}

func LookupUser(name string) int {
	mutex.Lock()
	defer mutex.Unlock()
	if u, ok := users[name]; ok {
		return u
	}
	var uid = -1
	if u, err := user.Lookup(name); err == nil {
		uid, _ = strconv.Atoi(u.Uid)
	} else {
		if g, e := strconv.Atoi(name); e == nil {
			uid = g
		} else {
			logger.Warnf("lookup user %s: %s", name, err)
		}
	}
	users[name] = uid
	return uid
}

func LookupGroup(name string) int {
	mutex.Lock()
	defer mutex.Unlock()
	if u, ok := groups[name]; ok {
		return u
	}
	var gid = -1
	if u, err := user.LookupGroup(name); err == nil {
		gid, _ = strconv.Atoi(u.Gid)
	} else {
		if g, e := strconv.Atoi(name); e == nil {
			gid = g
		} else {
			logger.Warnf("lookup group %s: %s", name, err)
		}
	}
	groups[name] = gid
	return gid
}