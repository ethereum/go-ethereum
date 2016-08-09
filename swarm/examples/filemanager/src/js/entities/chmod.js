(function(angular) {
    'use strict';
    angular.module('FileManagerApp').service('chmod', function () {

        var Chmod = function(initValue) {
            this.owner = this.getRwxObj();
            this.group = this.getRwxObj();
            this.others = this.getRwxObj();

            if (initValue) {
                var codes = isNaN(initValue) ?
                    this.convertfromCode(initValue):
                    this.convertfromOctal(initValue);

                if (! codes) {
                    throw new Error('Invalid chmod input data (%s)'.replace('%s', initValue));
                }

                this.owner = codes.owner;
                this.group = codes.group;
                this.others = codes.others;
            }
        };

        Chmod.prototype.toOctal = function(prepend, append) {
            var result = [];
            ['owner', 'group', 'others'].forEach(function(key, i) {
                result[i]  = this[key].read  && this.octalValues.read  || 0;
                result[i] += this[key].write && this.octalValues.write || 0;
                result[i] += this[key].exec  && this.octalValues.exec  || 0;
            }.bind(this));
            return (prepend||'') + result.join('') + (append||'');
        };

        Chmod.prototype.toCode = function(prepend, append) {
            var result = [];
            ['owner', 'group', 'others'].forEach(function(key, i) {
                result[i]  = this[key].read  && this.codeValues.read  || '-';
                result[i] += this[key].write && this.codeValues.write || '-';
                result[i] += this[key].exec  && this.codeValues.exec  || '-';
            }.bind(this));
            return (prepend||'') + result.join('') + (append||'');
        };

        Chmod.prototype.getRwxObj = function() {
            return {
                read: false,
                write: false,
                exec: false
            };
        };

        Chmod.prototype.octalValues = {
            read: 4, write: 2, exec: 1
        };

        Chmod.prototype.codeValues = {
            read: 'r', write: 'w', exec: 'x'
        };

        Chmod.prototype.convertfromCode = function (str) {
            str = ('' + str).replace(/\s/g, '');
            str = str.length === 10 ? str.substr(1) : str;
            if (! /^[-rwxts]{9}$/.test(str)) {
                return;
            }

            var result = [], vals = str.match(/.{1,3}/g);
            for (var i in vals) {
                var rwxObj = this.getRwxObj();
                rwxObj.read  = /r/.test(vals[i]);
                rwxObj.write = /w/.test(vals[i]);
                rwxObj.exec  = /x|t/.test(vals[i]);
                result.push(rwxObj);
            }

            return {
                owner : result[0],
                group : result[1],
                others: result[2]
            };
        };

        Chmod.prototype.convertfromOctal = function (str) {
            str = ('' + str).replace(/\s/g, '');
            str = str.length === 4 ? str.substr(1) : str;
            if (! /^[0-7]{3}$/.test(str)) {
                return;
            }

            var result = [], vals = str.match(/.{1}/g);
            for (var i in vals) {
                var rwxObj = this.getRwxObj();
                rwxObj.read  = /[4567]/.test(vals[i]);
                rwxObj.write = /[2367]/.test(vals[i]);
                rwxObj.exec  = /[1357]/.test(vals[i]);
                result.push(rwxObj);
            }

            return {
                owner : result[0],
                group : result[1],
                others: result[2]
            };
        };

        return Chmod;
    });
})(angular);