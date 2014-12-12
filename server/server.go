package main

import (
  "io"
  "log"
  "net"
  "os/exec"
  "syscall"

  "github.com/docker/libchan"
  "github.com/docker/libchan/spdy"
)

type RemoteCommand struct {
  Cmd        string
  Args       []string
  Stdin      io.Reader
  Stdout     io.WriteCloser
  Stderr     io.WriteCloser
  StatusChan libchan.Sender
}

type CommandResponse struct {
  Status int
}

func main() {
  var listener net.Listener
  var err error
  listener, err = net.Listen("tcp", "localhost:9323")

  if err != nil {
    log.Fatal(err)
  }

  tl, err := spdy.NewTransportListener(listener, spdy.NoAuthenticator)
  if err != nil {
    log.Fatal(err)
  }

  for {
    t, err := tl.AcceptTransport()
    if err != nil {
      log.Print(err)
      break
    }

    go func() {
      for {
        receiver, err := t.WaitReceiveChannel()
        if err != nil {
          log.Print(err)
          break
        }

        go func() {
          for {
            command := &RemoteCommand{}
            err := receiver.Receive(command)
            if err != nil {
              log.Print(err)
              break
            }

            cmd := exec.Command(command.Cmd, command.Args...)
            cmd.Stdout = command.Stdout
            cmd.Stderr = command.Stderr

            stdin, err := cmd.StdinPipe()
            if err != nil {
              log.Print(err)
              break
            }
            go func() {
              io.Copy(stdin, command.Stdin)
              stdin.Close()
            }()

            res := cmd.Run()
            command.Stdout.Close()
            command.Stderr.Close()
            returnResult := &CommandResponse{}
            if res != nil {
              if exiterr, ok := res.(*exec.ExitError); ok {
                returnResult.Status = exiterr.Sys().(syscall.WaitStatus).ExitStatus()
              } else {
                log.Print(res)
                returnResult.Status = 10
              }
            }

            err = command.StatusChan.Send(returnResult)
            if err != nil {
              log.Print(err)
            }
          }
        }()
      }
    }()
  }
}
