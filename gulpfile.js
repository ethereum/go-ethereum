#!/usr/bin/env node

'use strict';

var del = require('del');
var gulp = require('gulp');
var browserify = require('gulp-browserify-thin');
var jshint = require('gulp-jshint');
var uglify = require("gulp-uglify");
var rename = require("gulp-rename");
var bower = require('bower');

var DEST = './dist/';

gulp.task('bower', function(cb){
  bower.commands.install().on('end', function (installed){
    console.log(installed);
    cb();
  });
});

gulp.task('lint', function(){
  return gulp.src(['./*.js', './lib/*.js'])
    .pipe(jshint())
    .pipe(jshint.reporter('default'));
});

gulp.task('clean', ['lint'], function(cb) {
  del([ DEST ], cb);
});

gulp.task('build', ['clean'], function () {
  return browserify()
    .require('./index.js', { expose: 'web3'})
    .bundle('ethereum.js')
    .on('error', function(err)
    {
    	console.error(err.toString());
    	process.exit(1);
    })
    .pipe(gulp.dest( DEST ));
});

gulp.task('minify', ['build'], function(){
  return gulp.src( DEST + 'ethereum.js')
    .pipe(gulp.dest( DEST ))
    .pipe(uglify())
    .pipe(rename('ethereum.min.js'))
    .pipe(gulp.dest( DEST ));
});

gulp.task('watch', function() {
  gulp.watch(['./lib/*.js'], ['lint', 'build', 'minify']);
});

gulp.task('default', ['bower', 'lint', 'build', 'minify']);
