var spawn = require('win-spawn')
var P = require('autoresolve')
var testutil = require('testutil')
var colors = require('colors')

/* global describe, it, T, EQ */

describe('death', function() {
  describe('default behavior', function() {
    it('should catch SIGINT, SIGTERM, and SIGQUIT and return 3', function(done) {
      var signals = []
      var progPath = P('test/resources/default')
      var prog = spawn(progPath, [])
      //console.dir(prog)

      prog.stdout.on('data', function(data) {
        //console.log(colors.cyan(data.toString()))
      })

      prog.stderr.on('data', function(data) {
        //console.error(colors.red(data.toString()))
        signals = signals.concat(data.toString().trim().split('\n'))
      })

      prog.on('exit', function(code) {
        EQ (code, 3)
        //console.dir(signals)
        T (signals.indexOf('SIGQUIT') >= 0)
        T (signals.indexOf('SIGTERM') >= 0)
        T (signals.indexOf('SIGINT') >= 0)
        done()
      })

      setTimeout(function() {
        prog.kill('SIGINT')
        process.kill(prog.pid, 'SIGTERM')
        prog.kill('SIGQUIT')
      }, 100)

    })
  })

  describe('other signal', function() {
    it('should catch SIGINT, SIGTERM, SIGQUIT, and SIGHUP and return 4', function(done) {
      var signals = []
      var progPath = P('test/resources/sighup')
      var prog = spawn(progPath, [])
      //console.dir(prog)

      prog.stdout.on('data', function(data) {
        //console.log(colors.cyan(data.toString()))
      })

      prog.stderr.on('data', function(data) {
        //console.error(colors.red(data.toString()))
        signals = signals.concat(data.toString().trim().split('\n'))
      })

      prog.on('exit', function(code) {
        EQ (code, 4)
        //console.dir(signals)
        T (signals.indexOf('SIGQUIT') >= 0)
        T (signals.indexOf('SIGTERM') >= 0)
        T (signals.indexOf('SIGINT') >= 0)
        T (signals.indexOf('SIGHUP') >= 0)
        done()
      })

      setTimeout(function() {
        prog.kill('SIGINT')
        process.kill(prog.pid, 'SIGTERM')
        prog.kill('SIGQUIT')
        prog.kill('SIGHUP')
      }, 100)

    })
  })

  describe('disable signal', function() {
    it('should catch SIGINT and SIGTERM', function(done) {
      var signals = []
      var progPath = P('test/resources/disable')
      var prog = spawn(progPath, [])
      //console.dir(prog)

      prog.stdout.on('data', function(data) {
        //console.log(colors.cyan(data.toString()))
      })

      prog.stderr.on('data', function(data) {
        //console.error(colors.red(data.toString()))
        signals = signals.concat(data.toString().trim().split('\n'))
      })

      prog.on('exit', function(code) {
        T (signals.indexOf('SIGQUIT') < 0)
        T (signals.indexOf('SIGTERM') >= 0)
        T (signals.indexOf('SIGINT') >= 0)
        done()
      })

      setTimeout(function() {
        prog.kill('SIGINT')
        prog.kill('SIGTERM')
        setTimeout(function() {
          prog.kill('SIGQUIT') //this actually kills it since we disabled it
        },10)
      }, 100)

    })
  })

  describe('uncaughException', function() {
    describe('> when set to true', function() {
      it('should catch uncaughtException', function(done) {
        var errData = ''
        var progPath = P('test/resources/uncaughtException-true')
        var prog = spawn(progPath, [])
        //console.dir(prog)

        prog.stdout.on('data', function(data) {
          //console.log(colors.cyan(data.toString()))
        })

        prog.stderr.on('data', function(data) {
          //console.error(colors.red(data.toString()))
          errData += data.toString().trim()
        })

        prog.on('exit', function(code) {
          EQ (code, 70)
          T (errData.indexOf('uncaughtException') >= 0)
          T (errData.indexOf('UNCAUGHT SELF') >= 0)
          done()
        })
      })
    })

    describe('> when set to false', function() {
      it('should catch uncaughtException', function(done) {
        var errData = ''
        var progPath = P('test/resources/uncaughtException-false')
        var prog = spawn(progPath, [])
        //console.dir(prog)

        prog.stdout.on('data', function(data) {
          //console.log(colors.cyan(data.toString()))
        })

        prog.stderr.on('data', function(data) {
          //console.error(colors.red(data.toString()))
          errData += data.toString().trim()
        })

        prog.on('exit', function(code) {
          EQ (code, 1)
          T (errData.indexOf('CAUGHT: uncaughtException') < 0)
          T (errData.indexOf('UNCAUGHT SELF') >= 0)
          done()
        })
      })
    })
  })
})

