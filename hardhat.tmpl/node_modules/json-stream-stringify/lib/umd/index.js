(function (global, factory) {
  typeof exports === 'object' && typeof module !== 'undefined' ? factory(exports, require('stream')) :
  typeof define === 'function' && define.amd ? define(['exports', 'stream'], factory) :
  (global = typeof globalThis !== 'undefined' ? globalThis : global || self, factory(global.jsonStreamStringify = {}, global.stream));
})(this, (function (exports, stream) { 'use strict';

  function _defineProperty(obj, key, value) {
    if (key in obj) {
      Object.defineProperty(obj, key, {
        value: value,
        enumerable: true,
        configurable: true,
        writable: true
      });
    } else {
      obj[key] = value;
    }

    return obj;
  }

  var _global, _global$JSON;

  const rxEscapable = /[\\"\u0000-\u001f\u007f-\u009f\u00ad\u0600-\u0604\u070f\u17b4\u17b5\u200c-\u200f\u2028-\u202f\u2060-\u206f\ufeff\ufff0-\uffff]/g; // table of character substitutions

  const meta = {
    '\b': '\\b',
    '\t': '\\t',
    '\n': '\\n',
    '\f': '\\f',
    '\r': '\\r',
    '"': '\\"',
    '\\': '\\\\'
  };

  function isReadableStream(value) {
    return typeof value.read === 'function' && typeof value.pause === 'function' && typeof value.resume === 'function' && typeof value.pipe === 'function' && typeof value.once === 'function' && typeof value.removeListener === 'function';
  }

  var Types;

  (function (Types) {
    Types[Types["Array"] = 0] = "Array";
    Types[Types["Object"] = 1] = "Object";
    Types[Types["ReadableString"] = 2] = "ReadableString";
    Types[Types["ReadableObject"] = 3] = "ReadableObject";
    Types[Types["Primitive"] = 4] = "Primitive";
    Types[Types["Promise"] = 5] = "Promise";
  })(Types || (Types = {}));

  function getType(value) {
    if (!value) return Types.Primitive;
    if (typeof value.then === 'function') return Types.Promise;
    if (isReadableStream(value)) return value._readableState.objectMode ? Types.ReadableObject : Types.ReadableString;
    if (Array.isArray(value)) return Types.Array;
    if (typeof value === 'object' || value instanceof Object) return Types.Object;
    return Types.Primitive;
  }

  function escapeString(string) {
    // Modified code, original code by Douglas Crockford
    // Original: https://github.com/douglascrockford/JSON-js/blob/master/json2.js
    // If the string contains no control characters, no quote characters, and no
    // backslash characters, then we can safely slap some quotes around it.
    // Otherwise we must also replace the offending characters with safe escape
    // sequences.
    return string.replace(rxEscapable, a => {
      const c = meta[a];
      return typeof c === 'string' ? c : `\\u${a.charCodeAt(0).toString(16).padStart(4, '0')}`;
    });
  }

  let primitiveToJSON;

  if (((_global = global) === null || _global === void 0 ? void 0 : (_global$JSON = _global.JSON) === null || _global$JSON === void 0 ? void 0 : _global$JSON.stringify) instanceof Function) {
    try {
      if (JSON.stringify(global.BigInt ? global.BigInt('123') : '') !== '123') throw new Error();
      primitiveToJSON = JSON.stringify;
    } catch (err) {
      // Add support for bigint for primitiveToJSON
      // eslint-disable-next-line no-confusing-arrow
      primitiveToJSON = value => typeof value === 'bigint' ? String(value) : JSON.stringify(value);
    }
  } else {
    primitiveToJSON = value => {
      switch (typeof value) {
        case 'string':
          return `"${escapeString(value)}"`;

        case 'number':
          return Number.isFinite(value) ? String(value) : 'null';

        case 'bigint':
          return String(value);

        case 'boolean':
          return value ? 'true' : 'false';

        case 'object':
          if (!value) {
            return 'null';
          }

        // eslint-disable-next-line no-fallthrough

        default:
          // This should never happen, I can't imagine a situation where this executes.
          // If you find a way, please open a ticket or PR
          throw Object.assign(new Error(`Not a primitive "${typeof value}".`), {
            value
          });
      }
    };
  }
  /*
  function quoteString(string: string) {
    return primitiveToJSON(String(string));
  }
  */


  const cache = new Map();

  function quoteString(string) {
    const useCache = string.length < 10000; // eslint-disable-next-line no-lonely-if

    if (useCache && cache.has(string)) {
      return cache.get(string);
    }

    const str = primitiveToJSON(String(string));
    if (useCache) cache.set(string, str);
    return str;
  }

  function readAsPromised(stream, size) {
    var _stream$_readableStat;

    const value = stream.read(size);

    if (value === null && !(stream.readableEnded || (_stream$_readableStat = stream._readableState) !== null && _stream$_readableStat !== void 0 && _stream$_readableStat.ended)) {
      return new Promise((resolve, reject) => {
        const endListener = () => resolve(null);

        stream.once('end', endListener);
        stream.once('error', reject);
        stream.once('readable', () => {
          stream.removeListener('end', endListener);
          stream.removeListener('error', reject);
          readAsPromised(stream, size).then(resolve, reject);
        });
      });
    }

    return Promise.resolve(value);
  }

  var ReadState;

  (function (ReadState) {
    ReadState[ReadState["Inactive"] = 0] = "Inactive";
    ReadState[ReadState["Reading"] = 1] = "Reading";
    ReadState[ReadState["ReadMore"] = 2] = "ReadMore";
    ReadState[ReadState["Consumed"] = 3] = "Consumed";
  })(ReadState || (ReadState = {}));

  class JsonStreamStringify extends stream.Readable {
    constructor(input, replacer, spaces, cycle = false, bufferSize = 512) {
      super({
        encoding: 'utf8'
      });

      _defineProperty(this, "cycle", void 0);

      _defineProperty(this, "bufferSize", void 0);

      _defineProperty(this, "item", void 0);

      _defineProperty(this, "indent", void 0);

      _defineProperty(this, "root", void 0);

      _defineProperty(this, "include", void 0);

      _defineProperty(this, "replacer", void 0);

      _defineProperty(this, "visited", void 0);

      _defineProperty(this, "objectItem", void 0);

      _defineProperty(this, "buffer", '');

      _defineProperty(this, "bufferLength", 0);

      _defineProperty(this, "pushCalled", false);

      _defineProperty(this, "readSize", 0);

      _defineProperty(this, "prePush", void 0);

      _defineProperty(this, "readState", ReadState.Inactive);

      this.cycle = cycle;
      this.bufferSize = bufferSize;
      const spaceType = typeof spaces;

      if (spaceType === 'number') {
        this.indent = ' '.repeat(spaces);
      } else if (spaceType === 'string') {
        this.indent = spaces;
      }

      const replacerType = typeof replacer;

      if (replacerType === 'object') {
        this.include = replacer;
      } else if (replacerType === 'function') {
        this.replacer = replacer;
      }

      this.visited = cycle ? new WeakMap() : [];
      this.root = {
        value: {
          '': input
        },
        depth: 0,
        indent: '',
        path: []
      };
      this.setItem(input, this.root, '');
    }

    setItem(value, parent, key = '') {
      // call toJSON where applicable
      if (value && typeof value === 'object' && typeof value.toJSON === 'function') {
        value = value.toJSON(key);
      } // use replacer if applicable


      if (this.replacer) {
        value = this.replacer.call(parent.value, key, value);
      } // coerece functions and symbols into undefined


      if (value instanceof Function || typeof value === 'symbol') {
        value = undefined;
      }

      const type = getType(value);
      let path; // check for circular structure

      if (!this.cycle && type !== Types.Primitive) {
        if (this.visited.some(v => v === value)) {
          this.destroy(Object.assign(new Error('Converting circular structure to JSON'), {
            value,
            key
          }));
          return;
        }

        this.visited.push(value);
      } else if (this.cycle && type !== Types.Primitive) {
        path = this.visited.get(value);

        if (path) {
          this._push(`{"$ref":"$${path.map(v => `[${Number.isInteger(v) ? v : escapeString(quoteString(v))}]`).join('')}"}`);

          this.item = parent;
          return;
        }

        path = parent === this.root ? [] : parent.path.concat(key);
        this.visited.set(value, path);
      }

      if (type === Types.Object) {
        this.setObjectItem(value, parent);
      } else if (type === Types.Array) {
        this.setArrayItem(value, parent);
      } else if (type === Types.Primitive) {
        if (parent !== this.root && typeof key === 'string') {
          // (<any>parent).write(key, primitiveToJSON(value));
          if (value === undefined) ; else {
            this._push(primitiveToJSON(value));
          } // undefined values in objects should be rejected

        } else if (value === undefined && typeof key === 'number') {
          // undefined values in array should be null
          this._push('null');
        } else if (value === undefined) ; else {
          this._push(primitiveToJSON(value));
        }

        this.item = parent;
        return;
      } else if (type === Types.Promise) {
        this.setPromiseItem(value, parent, key);
      } else if (type === Types.ReadableString) {
        this.setReadableStringItem(value, parent);
      } else if (type === Types.ReadableObject) {
        this.setReadableObjectItem(value, parent);
      }

      this.item.value = value;
      this.item.depth = parent.depth + 1;
      if (this.indent) this.item.indent = this.indent.repeat(this.item.depth);
      this.item.path = path;
    }

    setReadableStringItem(input, parent) {
      var _input$_readableState, _input$_readableState2;

      if (input.readableEnded || (_input$_readableState = input._readableState) !== null && _input$_readableState !== void 0 && _input$_readableState.endEmitted) {
        this.emit('error', new Error('Readable Stream has ended before it was serialized. All stream data have been lost'), input, parent.path);
      } else if (input.readableFlowing || (_input$_readableState2 = input._readableState) !== null && _input$_readableState2 !== void 0 && _input$_readableState2.flowing) {
        input.pause();
        this.emit('error', new Error('Readable Stream is in flowing mode, data may have been lost. Trying to pause stream.'), input, parent.path);
      }

      const that = this;
      this.prePush = '"';
      this.item = {
        type: 'readable string',

        async read(size) {
          try {
            const data = await readAsPromised(input, size);

            if (data === null) {
              that._push('"');

              that.item = parent;
              that.unvisit(input);
              return;
            }

            if (data) that._push(escapeString(data.toString()));
          } catch (err) {
            that.emit('error', err);
            that.destroy();
          }
        }

      };
    }

    setReadableObjectItem(input, parent) {
      var _input$_readableState3, _input$_readableState4;

      if (input.readableEnded || (_input$_readableState3 = input._readableState) !== null && _input$_readableState3 !== void 0 && _input$_readableState3.endEmitted) {
        this.emit('error', new Error('Readable Stream has ended before it was serialized. All stream data have been lost'), input, parent.path);
      } else if (input.readableFlowing || (_input$_readableState4 = input._readableState) !== null && _input$_readableState4 !== void 0 && _input$_readableState4.flowing) {
        input.pause();
        this.emit('error', new Error('Readable Stream is in flowing mode, data may have been lost. Trying to pause stream.'), input, parent.path);
      }

      const that = this;

      this._push('[');

      let first = true;
      let i = 0;
      const item = {
        type: 'readable object',

        async read() {
          try {
            let out = '';
            const data = await readAsPromised(input);

            if (data === null) {
              if (i && that.indent) {
                out += `\n${parent.indent}`;
              }

              out += ']';

              that._push(out);

              that.item = parent;
              that.unvisit(input);
              return;
            }

            if (first) first = false;else out += ',';
            if (that.indent) out += `\n${item.indent}`;
            that.prePush = out;
            that.setItem(data, item, i);
            i += 1;
          } catch (err) {
            that.emit('error', err);
            that.destroy();
          }
        }

      };
      this.item = item;
    }

    setPromiseItem(input, parent, key) {
      const that = this;
      let read = false;
      this.item = {
        async read() {
          if (read) return;

          try {
            read = true;
            that.setItem(await input, parent, key);
          } catch (err) {
            that.emit('error', err);
            that.destroy();
          }
        }

      };
    }

    setArrayItem(input, parent) {
      // const entries = input.slice().reverse();
      let i = 0;
      const len = input.length;
      let first = true;
      const that = this;
      const item = {
        read() {
          let out = '';
          let wasFirst = false;

          if (first) {
            first = false;
            wasFirst = true;

            if (!len) {
              that._push('[]');

              that.unvisit(input);
              that.item = parent;
              return;
            }

            out += '[';
          }

          const entry = input[i];

          if (i === len) {
            if (that.indent) out += `\n${parent.indent}`;
            out += ']';

            that._push(out);

            that.item = parent;
            that.unvisit(input);
            return;
          }

          if (!wasFirst) out += ',';
          if (that.indent) out += `\n${item.indent}`;

          that._push(out);

          that.setItem(entry, item, i);
          i += 1;
        }

      };
      this.item = item;
    }

    unvisit(item) {
      if (this.cycle) return;

      const _i = this.visited.indexOf(item);

      if (_i > -1) this.visited.splice(_i, 1);
    }

    setObjectItem(input, parent = undefined) {
      const keys = Object.keys(input);
      let i = 0;
      const len = keys.length;
      let first = true;
      const that = this;
      const {
        include
      } = this;
      let hasItems = false;
      let key;
      const item = {
        read() {
          var _include$indexOf;

          if (i === 0) that._push('{');

          if (i === len) {
            that.objectItem = undefined;

            if (!hasItems) {
              that._push('}');
            } else {
              that._push(`${that.indent ? `\n${parent.indent}` : ''}}`);
            }

            that.item = parent;
            that.unvisit(input);
            return;
          }

          key = keys[i];

          if ((include === null || include === void 0 ? void 0 : (_include$indexOf = include.indexOf) === null || _include$indexOf === void 0 ? void 0 : _include$indexOf.call(include, key)) === -1) {
            // replacer array excludes this key
            i += 1;
            return;
          }

          that.objectItem = item;
          i += 1;
          that.setItem(input[key], item, key);
        },

        write() {
          const out = `${hasItems && !first ? ',' : ''}${item.indent ? `\n${item.indent}` : ''}${quoteString(key)}:${that.indent ? ' ' : ''}`;
          first = false;
          hasItems = true;
          that.objectItem = undefined;
          return out;
        }

      };
      this.item = item;
    }

    _push(data) {
      const out = (this.objectItem ? this.objectItem.write() : '') + data;

      if (this.prePush && out.length) {
        this.buffer += this.prePush;
        this.prePush = undefined;
      }

      this.buffer += out;

      if (this.buffer.length >= this.bufferSize) {
        this.pushCalled = !this.push(this.buffer);
        this.buffer = '';
        this.bufferLength = 0;
        return false;
      }

      return true;
    }

    async _read(size) {
      if (this.readState === ReadState.Consumed) return;

      if (this.readState !== ReadState.Inactive) {
        this.readState = ReadState.ReadMore;
        return;
      }

      this.readState = ReadState.Reading;
      this.pushCalled = false;
      let p;

      while (!this.pushCalled && this.item !== this.root && this.buffer !== undefined) {
        p = this.item.read(size); // eslint-disable-next-line no-await-in-loop

        if (p) await p;
      }

      if (this.buffer === undefined) return;

      if (this.item === this.root) {
        if (this.buffer.length) this.push(this.buffer);
        this.push(null);
        this.readState = ReadState.Consumed;
        this.cleanup();
        return;
      }

      if (this.readState === ReadState.ReadMore) {
        this.readState = ReadState.Inactive;
        await this._read(size);
        return;
      }

      this.readState = ReadState.Inactive;
    }

    cleanup() {
      this.readState = ReadState.Consumed;
      this.buffer = undefined;
      this.visited = undefined;
      this.item = undefined;
      this.root = undefined;
      this.prePush = undefined;
    }

    destroy(error) {
      var _super$destroy;

      if (error) this.emit('error', error);
      (_super$destroy = super.destroy) === null || _super$destroy === void 0 ? void 0 : _super$destroy.call(this);
      this.cleanup();
      return this;
    }

  }

  exports.JsonStreamStringify = JsonStreamStringify;

  Object.defineProperty(exports, '__esModule', { value: true });

}));
//# sourceMappingURL=index.js.map
