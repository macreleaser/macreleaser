package logging

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestBulletFormatterAction(t *testing.T) {
	f := &BulletFormatter{}
	entry := &logrus.Entry{
		Level: logrus.InfoLevel,
		Data:  logrus.Fields{"action": "building project"},
	}
	out, err := f.Format(entry)
	if err != nil {
		t.Fatal(err)
	}
	want := "  * building project\n"
	if string(out) != want {
		t.Errorf("got %q, want %q", string(out), want)
	}
}

func TestBulletFormatterActionWithFields(t *testing.T) {
	f := &BulletFormatter{}
	entry := &logrus.Entry{
		Level: logrus.InfoLevel,
		Data: logrus.Fields{
			"action": "git state",
			"commit": "4cb72c9",
			"branch": "main",
		},
	}
	out, err := f.Format(entry)
	if err != nil {
		t.Fatal(err)
	}
	want := "  * git state  branch=main commit=4cb72c9\n"
	if string(out) != want {
		t.Errorf("got %q, want %q", string(out), want)
	}
}

func TestBulletFormatterInfo(t *testing.T) {
	f := &BulletFormatter{}
	entry := &logrus.Entry{
		Level:   logrus.InfoLevel,
		Message: "scheme=TestApp configuration=Release",
		Data:    logrus.Fields{},
	}
	out, err := f.Format(entry)
	if err != nil {
		t.Fatal(err)
	}
	want := "    * scheme=TestApp configuration=Release\n"
	if string(out) != want {
		t.Errorf("got %q, want %q", string(out), want)
	}
}

func TestBulletFormatterWarn(t *testing.T) {
	f := &BulletFormatter{}
	entry := &logrus.Entry{
		Level:   logrus.WarnLevel,
		Message: "some warning",
		Data:    logrus.Fields{},
	}
	out, err := f.Format(entry)
	if err != nil {
		t.Fatal(err)
	}
	want := "    ! some warning\n"
	if string(out) != want {
		t.Errorf("got %q, want %q", string(out), want)
	}
}

func TestBulletFormatterError(t *testing.T) {
	f := &BulletFormatter{}
	entry := &logrus.Entry{
		Level:   logrus.ErrorLevel,
		Message: "build failed",
		Data:    logrus.Fields{},
	}
	out, err := f.Format(entry)
	if err != nil {
		t.Fatal(err)
	}
	want := "  x build failed\n"
	if string(out) != want {
		t.Errorf("got %q, want %q", string(out), want)
	}
}
