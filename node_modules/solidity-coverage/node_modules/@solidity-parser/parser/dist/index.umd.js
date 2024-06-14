"use strict";
(function universalModuleDefinition(root, factory) {
  if(typeof exports === 'object' && typeof module === 'object')
    module.exports = factory();
  else if(typeof define === 'function' && define.amd)
    define([], factory);
  else if(typeof exports === 'object')
    exports["SolidityParser"] = factory();
  else
    root["SolidityParser"] = factory();
})(
typeof globalThis !== 'undefined' ? globalThis
: typeof global !== 'undefined' ? global
: typeof self !== 'undefined' ? self
: this || {}
, () => {
var SolidityParser = (() => {
  var __defProp = Object.defineProperty;
  var __export = (target, all) => {
    for (var name in all)
      __defProp(target, name, {get: all[name], enumerable: true});
  };

  // src/index.ts
  var src_exports = {};
  __export(src_exports, {
    ParserError: () => ParserError,
    parse: () => parse,
    tokenize: () => tokenize,
    visit: () => visit
  });

  // node_modules/antlr4/dist/antlr4.web.mjs
  var t = {92: () => {
  }};
  var e = {};
  function n(s2) {
    var i2 = e[s2];
    if (i2 !== void 0)
      return i2.exports;
    var r2 = e[s2] = {exports: {}};
    return t[s2](r2, r2.exports, n), r2.exports;
  }
  n.d = (t2, e2) => {
    for (var s2 in e2)
      n.o(e2, s2) && !n.o(t2, s2) && Object.defineProperty(t2, s2, {enumerable: true, get: e2[s2]});
  }, n.o = (t2, e2) => Object.prototype.hasOwnProperty.call(t2, e2);
  var s = {};
  (() => {
    n.d(s, {dx: () => $, q2: () => Lt, FO: () => Ce, xf: () => Ie, Gy: () => ve, s4: () => Pt, c7: () => be, _7: () => le, tx: () => Ae, gp: () => me, cK: () => Ot, zs: () => Te, AV: () => Ie, Xp: () => m2, VS: () => E2, ul: () => W, hW: () => Ut, x1: () => Xt, z5: () => ee, oN: () => de, TB: () => pe, u1: () => ge, _b: () => Fe, $F: () => se, _T: () => Be, db: () => ie, Zx: () => te, _x: () => Ft, r8: () => M2, JI: () => w2, TP: () => P2, WU: () => t2, Nj: () => c2, ZP: () => We});
    class t2 {
      constructor() {
        this.source = null, this.type = null, this.channel = null, this.start = null, this.stop = null, this.tokenIndex = null, this.line = null, this.column = null, this._text = null;
      }
      getTokenSource() {
        return this.source[0];
      }
      getInputStream() {
        return this.source[1];
      }
      get text() {
        return this._text;
      }
      set text(t3) {
        this._text = t3;
      }
    }
    function e2(t3, e3) {
      if (!Array.isArray(t3) || !Array.isArray(e3))
        return false;
      if (t3 === e3)
        return true;
      if (t3.length !== e3.length)
        return false;
      for (let n2 = 0; n2 < t3.length; n2++)
        if (!(t3[n2] === e3[n2] || t3[n2].equals && t3[n2].equals(e3[n2])))
          return false;
      return true;
    }
    t2.INVALID_TYPE = 0, t2.EPSILON = -2, t2.MIN_USER_TOKEN_TYPE = 1, t2.EOF = -1, t2.DEFAULT_CHANNEL = 0, t2.HIDDEN_CHANNEL = 1;
    const i2 = Math.round(Math.random() * Math.pow(2, 32));
    function r2(t3) {
      if (!t3)
        return 0;
      const e3 = typeof t3, n2 = e3 === "string" ? t3 : !(e3 !== "object" || !t3.toString) && t3.toString();
      if (!n2)
        return 0;
      let s2, r3;
      const o3 = 3 & n2.length, a3 = n2.length - o3;
      let l3 = i2;
      const h3 = 3432918353, c3 = 461845907;
      let u3 = 0;
      for (; u3 < a3; )
        r3 = 255 & n2.charCodeAt(u3) | (255 & n2.charCodeAt(++u3)) << 8 | (255 & n2.charCodeAt(++u3)) << 16 | (255 & n2.charCodeAt(++u3)) << 24, ++u3, r3 = (65535 & r3) * h3 + (((r3 >>> 16) * h3 & 65535) << 16) & 4294967295, r3 = r3 << 15 | r3 >>> 17, r3 = (65535 & r3) * c3 + (((r3 >>> 16) * c3 & 65535) << 16) & 4294967295, l3 ^= r3, l3 = l3 << 13 | l3 >>> 19, s2 = 5 * (65535 & l3) + ((5 * (l3 >>> 16) & 65535) << 16) & 4294967295, l3 = 27492 + (65535 & s2) + ((58964 + (s2 >>> 16) & 65535) << 16);
      switch (r3 = 0, o3) {
        case 3:
          r3 ^= (255 & n2.charCodeAt(u3 + 2)) << 16;
        case 2:
          r3 ^= (255 & n2.charCodeAt(u3 + 1)) << 8;
        case 1:
          r3 ^= 255 & n2.charCodeAt(u3), r3 = (65535 & r3) * h3 + (((r3 >>> 16) * h3 & 65535) << 16) & 4294967295, r3 = r3 << 15 | r3 >>> 17, r3 = (65535 & r3) * c3 + (((r3 >>> 16) * c3 & 65535) << 16) & 4294967295, l3 ^= r3;
      }
      return l3 ^= n2.length, l3 ^= l3 >>> 16, l3 = 2246822507 * (65535 & l3) + ((2246822507 * (l3 >>> 16) & 65535) << 16) & 4294967295, l3 ^= l3 >>> 13, l3 = 3266489909 * (65535 & l3) + ((3266489909 * (l3 >>> 16) & 65535) << 16) & 4294967295, l3 ^= l3 >>> 16, l3 >>> 0;
    }
    class o2 {
      constructor() {
        this.count = 0, this.hash = 0;
      }
      update() {
        for (let t3 = 0; t3 < arguments.length; t3++) {
          const e3 = arguments[t3];
          if (e3 != null)
            if (Array.isArray(e3))
              this.update.apply(this, e3);
            else {
              let t4 = 0;
              switch (typeof e3) {
                case "undefined":
                case "function":
                  continue;
                case "number":
                case "boolean":
                  t4 = e3;
                  break;
                case "string":
                  t4 = r2(e3);
                  break;
                default:
                  e3.updateHashCode ? e3.updateHashCode(this) : console.log("No updateHashCode for " + e3.toString());
                  continue;
              }
              t4 *= 3432918353, t4 = t4 << 15 | t4 >>> 17, t4 *= 461845907, this.count = this.count + 1;
              let n2 = this.hash ^ t4;
              n2 = n2 << 13 | n2 >>> 19, n2 = 5 * n2 + 3864292196, this.hash = n2;
            }
        }
      }
      finish() {
        let t3 = this.hash ^ 4 * this.count;
        return t3 ^= t3 >>> 16, t3 *= 2246822507, t3 ^= t3 >>> 13, t3 *= 3266489909, t3 ^= t3 >>> 16, t3;
      }
      static hashStuff() {
        const t3 = new o2();
        return t3.update.apply(t3, arguments), t3.finish();
      }
    }
    function a2(t3) {
      return t3 ? typeof t3 == "string" ? r2(t3) : t3.hashCode() : -1;
    }
    function l2(t3, e3) {
      return t3 ? t3.equals(e3) : t3 === e3;
    }
    function h2(t3) {
      return t3 === null ? "null" : t3;
    }
    function c2(t3) {
      return Array.isArray(t3) ? "[" + t3.map(h2).join(", ") + "]" : "null";
    }
    const u2 = "h-";
    class d2 {
      constructor(t3, e3) {
        this.data = {}, this.hashFunction = t3 || a2, this.equalsFunction = e3 || l2;
      }
      add(t3) {
        const e3 = u2 + this.hashFunction(t3);
        if (e3 in this.data) {
          const n2 = this.data[e3];
          for (let e4 = 0; e4 < n2.length; e4++)
            if (this.equalsFunction(t3, n2[e4]))
              return n2[e4];
          return n2.push(t3), t3;
        }
        return this.data[e3] = [t3], t3;
      }
      has(t3) {
        return this.get(t3) != null;
      }
      get(t3) {
        const e3 = u2 + this.hashFunction(t3);
        if (e3 in this.data) {
          const n2 = this.data[e3];
          for (let e4 = 0; e4 < n2.length; e4++)
            if (this.equalsFunction(t3, n2[e4]))
              return n2[e4];
        }
        return null;
      }
      values() {
        return Object.keys(this.data).filter((t3) => t3.startsWith(u2)).flatMap((t3) => this.data[t3], this);
      }
      toString() {
        return c2(this.values());
      }
      get length() {
        return Object.keys(this.data).filter((t3) => t3.startsWith(u2)).map((t3) => this.data[t3].length, this).reduce((t3, e3) => t3 + e3, 0);
      }
    }
    class p2 {
      hashCode() {
        const t3 = new o2();
        return this.updateHashCode(t3), t3.finish();
      }
      evaluate(t3, e3) {
      }
      evalPrecedence(t3, e3) {
        return this;
      }
      static andContext(t3, e3) {
        if (t3 === null || t3 === p2.NONE)
          return e3;
        if (e3 === null || e3 === p2.NONE)
          return t3;
        const n2 = new g2(t3, e3);
        return n2.opnds.length === 1 ? n2.opnds[0] : n2;
      }
      static orContext(t3, e3) {
        if (t3 === null)
          return e3;
        if (e3 === null)
          return t3;
        if (t3 === p2.NONE || e3 === p2.NONE)
          return p2.NONE;
        const n2 = new f2(t3, e3);
        return n2.opnds.length === 1 ? n2.opnds[0] : n2;
      }
    }
    class g2 extends p2 {
      constructor(t3, e3) {
        super();
        const n2 = new d2();
        t3 instanceof g2 ? t3.opnds.map(function(t4) {
          n2.add(t4);
        }) : n2.add(t3), e3 instanceof g2 ? e3.opnds.map(function(t4) {
          n2.add(t4);
        }) : n2.add(e3);
        const s2 = x2(n2);
        if (s2.length > 0) {
          let t4 = null;
          s2.map(function(e4) {
            (t4 === null || e4.precedence < t4.precedence) && (t4 = e4);
          }), n2.add(t4);
        }
        this.opnds = Array.from(n2.values());
      }
      equals(t3) {
        return this === t3 || t3 instanceof g2 && e2(this.opnds, t3.opnds);
      }
      updateHashCode(t3) {
        t3.update(this.opnds, "AND");
      }
      evaluate(t3, e3) {
        for (let n2 = 0; n2 < this.opnds.length; n2++)
          if (!this.opnds[n2].evaluate(t3, e3))
            return false;
        return true;
      }
      evalPrecedence(t3, e3) {
        let n2 = false;
        const s2 = [];
        for (let i4 = 0; i4 < this.opnds.length; i4++) {
          const r3 = this.opnds[i4], o3 = r3.evalPrecedence(t3, e3);
          if (n2 |= o3 !== r3, o3 === null)
            return null;
          o3 !== p2.NONE && s2.push(o3);
        }
        if (!n2)
          return this;
        if (s2.length === 0)
          return p2.NONE;
        let i3 = null;
        return s2.map(function(t4) {
          i3 = i3 === null ? t4 : p2.andContext(i3, t4);
        }), i3;
      }
      toString() {
        const t3 = this.opnds.map((t4) => t4.toString());
        return (t3.length > 3 ? t3.slice(3) : t3).join("&&");
      }
    }
    class f2 extends p2 {
      constructor(t3, e3) {
        super();
        const n2 = new d2();
        t3 instanceof f2 ? t3.opnds.map(function(t4) {
          n2.add(t4);
        }) : n2.add(t3), e3 instanceof f2 ? e3.opnds.map(function(t4) {
          n2.add(t4);
        }) : n2.add(e3);
        const s2 = x2(n2);
        if (s2.length > 0) {
          const t4 = s2.sort(function(t5, e5) {
            return t5.compareTo(e5);
          }), e4 = t4[t4.length - 1];
          n2.add(e4);
        }
        this.opnds = Array.from(n2.values());
      }
      equals(t3) {
        return this === t3 || t3 instanceof f2 && e2(this.opnds, t3.opnds);
      }
      updateHashCode(t3) {
        t3.update(this.opnds, "OR");
      }
      evaluate(t3, e3) {
        for (let n2 = 0; n2 < this.opnds.length; n2++)
          if (this.opnds[n2].evaluate(t3, e3))
            return true;
        return false;
      }
      evalPrecedence(t3, e3) {
        let n2 = false;
        const s2 = [];
        for (let i3 = 0; i3 < this.opnds.length; i3++) {
          const r3 = this.opnds[i3], o3 = r3.evalPrecedence(t3, e3);
          if (n2 |= o3 !== r3, o3 === p2.NONE)
            return p2.NONE;
          o3 !== null && s2.push(o3);
        }
        if (!n2)
          return this;
        if (s2.length === 0)
          return null;
        return s2.map(function(t4) {
          return t4;
        }), null;
      }
      toString() {
        const t3 = this.opnds.map((t4) => t4.toString());
        return (t3.length > 3 ? t3.slice(3) : t3).join("||");
      }
    }
    function x2(t3) {
      const e3 = [];
      return t3.values().map(function(t4) {
        t4 instanceof p2.PrecedencePredicate && e3.push(t4);
      }), e3;
    }
    function T2(t3, e3) {
      if (t3 === null) {
        const t4 = {state: null, alt: null, context: null, semanticContext: null};
        return e3 && (t4.reachesIntoOuterContext = 0), t4;
      }
      {
        const n2 = {};
        return n2.state = t3.state || null, n2.alt = t3.alt === void 0 ? null : t3.alt, n2.context = t3.context || null, n2.semanticContext = t3.semanticContext || null, e3 && (n2.reachesIntoOuterContext = t3.reachesIntoOuterContext || 0, n2.precedenceFilterSuppressed = t3.precedenceFilterSuppressed || false), n2;
      }
    }
    class S2 {
      constructor(t3, e3) {
        this.checkContext(t3, e3), t3 = T2(t3), e3 = T2(e3, true), this.state = t3.state !== null ? t3.state : e3.state, this.alt = t3.alt !== null ? t3.alt : e3.alt, this.context = t3.context !== null ? t3.context : e3.context, this.semanticContext = t3.semanticContext !== null ? t3.semanticContext : e3.semanticContext !== null ? e3.semanticContext : p2.NONE, this.reachesIntoOuterContext = e3.reachesIntoOuterContext, this.precedenceFilterSuppressed = e3.precedenceFilterSuppressed;
      }
      checkContext(t3, e3) {
        t3.context !== null && t3.context !== void 0 || e3 !== null && e3.context !== null && e3.context !== void 0 || (this.context = null);
      }
      hashCode() {
        const t3 = new o2();
        return this.updateHashCode(t3), t3.finish();
      }
      updateHashCode(t3) {
        t3.update(this.state.stateNumber, this.alt, this.context, this.semanticContext);
      }
      equals(t3) {
        return this === t3 || t3 instanceof S2 && this.state.stateNumber === t3.state.stateNumber && this.alt === t3.alt && (this.context === null ? t3.context === null : this.context.equals(t3.context)) && this.semanticContext.equals(t3.semanticContext) && this.precedenceFilterSuppressed === t3.precedenceFilterSuppressed;
      }
      hashCodeForConfigSet() {
        const t3 = new o2();
        return t3.update(this.state.stateNumber, this.alt, this.semanticContext), t3.finish();
      }
      equalsForConfigSet(t3) {
        return this === t3 || t3 instanceof S2 && this.state.stateNumber === t3.state.stateNumber && this.alt === t3.alt && this.semanticContext.equals(t3.semanticContext);
      }
      toString() {
        return "(" + this.state + "," + this.alt + (this.context !== null ? ",[" + this.context.toString() + "]" : "") + (this.semanticContext !== p2.NONE ? "," + this.semanticContext.toString() : "") + (this.reachesIntoOuterContext > 0 ? ",up=" + this.reachesIntoOuterContext : "") + ")";
      }
    }
    class m2 {
      constructor(t3, e3) {
        this.start = t3, this.stop = e3;
      }
      clone() {
        return new m2(this.start, this.stop);
      }
      contains(t3) {
        return t3 >= this.start && t3 < this.stop;
      }
      toString() {
        return this.start === this.stop - 1 ? this.start.toString() : this.start.toString() + ".." + (this.stop - 1).toString();
      }
      get length() {
        return this.stop - this.start;
      }
    }
    m2.INVALID_INTERVAL = new m2(-1, -2);
    class E2 {
      constructor() {
        this.intervals = null, this.readOnly = false;
      }
      first(e3) {
        return this.intervals === null || this.intervals.length === 0 ? t2.INVALID_TYPE : this.intervals[0].start;
      }
      addOne(t3) {
        this.addInterval(new m2(t3, t3 + 1));
      }
      addRange(t3, e3) {
        this.addInterval(new m2(t3, e3 + 1));
      }
      addInterval(t3) {
        if (this.intervals === null)
          this.intervals = [], this.intervals.push(t3.clone());
        else {
          for (let e3 = 0; e3 < this.intervals.length; e3++) {
            const n2 = this.intervals[e3];
            if (t3.stop < n2.start)
              return void this.intervals.splice(e3, 0, t3);
            if (t3.stop === n2.start)
              return void (this.intervals[e3] = new m2(t3.start, n2.stop));
            if (t3.start <= n2.stop)
              return this.intervals[e3] = new m2(Math.min(n2.start, t3.start), Math.max(n2.stop, t3.stop)), void this.reduce(e3);
          }
          this.intervals.push(t3.clone());
        }
      }
      addSet(t3) {
        return t3.intervals !== null && t3.intervals.forEach((t4) => this.addInterval(t4), this), this;
      }
      reduce(t3) {
        if (t3 < this.intervals.length - 1) {
          const e3 = this.intervals[t3], n2 = this.intervals[t3 + 1];
          e3.stop >= n2.stop ? (this.intervals.splice(t3 + 1, 1), this.reduce(t3)) : e3.stop >= n2.start && (this.intervals[t3] = new m2(e3.start, n2.stop), this.intervals.splice(t3 + 1, 1));
        }
      }
      complement(t3, e3) {
        const n2 = new E2();
        return n2.addInterval(new m2(t3, e3 + 1)), this.intervals !== null && this.intervals.forEach((t4) => n2.removeRange(t4)), n2;
      }
      contains(t3) {
        if (this.intervals === null)
          return false;
        for (let e3 = 0; e3 < this.intervals.length; e3++)
          if (this.intervals[e3].contains(t3))
            return true;
        return false;
      }
      removeRange(t3) {
        if (t3.start === t3.stop - 1)
          this.removeOne(t3.start);
        else if (this.intervals !== null) {
          let e3 = 0;
          for (let n2 = 0; n2 < this.intervals.length; n2++) {
            const n3 = this.intervals[e3];
            if (t3.stop <= n3.start)
              return;
            if (t3.start > n3.start && t3.stop < n3.stop) {
              this.intervals[e3] = new m2(n3.start, t3.start);
              const s2 = new m2(t3.stop, n3.stop);
              return void this.intervals.splice(e3, 0, s2);
            }
            t3.start <= n3.start && t3.stop >= n3.stop ? (this.intervals.splice(e3, 1), e3 -= 1) : t3.start < n3.stop ? this.intervals[e3] = new m2(n3.start, t3.start) : t3.stop < n3.stop && (this.intervals[e3] = new m2(t3.stop, n3.stop)), e3 += 1;
          }
        }
      }
      removeOne(t3) {
        if (this.intervals !== null)
          for (let e3 = 0; e3 < this.intervals.length; e3++) {
            const n2 = this.intervals[e3];
            if (t3 < n2.start)
              return;
            if (t3 === n2.start && t3 === n2.stop - 1)
              return void this.intervals.splice(e3, 1);
            if (t3 === n2.start)
              return void (this.intervals[e3] = new m2(n2.start + 1, n2.stop));
            if (t3 === n2.stop - 1)
              return void (this.intervals[e3] = new m2(n2.start, n2.stop - 1));
            if (t3 < n2.stop - 1) {
              const s2 = new m2(n2.start, t3);
              return n2.start = t3 + 1, void this.intervals.splice(e3, 0, s2);
            }
          }
      }
      toString(t3, e3, n2) {
        return t3 = t3 || null, e3 = e3 || null, n2 = n2 || false, this.intervals === null ? "{}" : t3 !== null || e3 !== null ? this.toTokenString(t3, e3) : n2 ? this.toCharString() : this.toIndexString();
      }
      toCharString() {
        const e3 = [];
        for (let n2 = 0; n2 < this.intervals.length; n2++) {
          const s2 = this.intervals[n2];
          s2.stop === s2.start + 1 ? s2.start === t2.EOF ? e3.push("<EOF>") : e3.push("'" + String.fromCharCode(s2.start) + "'") : e3.push("'" + String.fromCharCode(s2.start) + "'..'" + String.fromCharCode(s2.stop - 1) + "'");
        }
        return e3.length > 1 ? "{" + e3.join(", ") + "}" : e3[0];
      }
      toIndexString() {
        const e3 = [];
        for (let n2 = 0; n2 < this.intervals.length; n2++) {
          const s2 = this.intervals[n2];
          s2.stop === s2.start + 1 ? s2.start === t2.EOF ? e3.push("<EOF>") : e3.push(s2.start.toString()) : e3.push(s2.start.toString() + ".." + (s2.stop - 1).toString());
        }
        return e3.length > 1 ? "{" + e3.join(", ") + "}" : e3[0];
      }
      toTokenString(t3, e3) {
        const n2 = [];
        for (let s2 = 0; s2 < this.intervals.length; s2++) {
          const i3 = this.intervals[s2];
          for (let s3 = i3.start; s3 < i3.stop; s3++)
            n2.push(this.elementName(t3, e3, s3));
        }
        return n2.length > 1 ? "{" + n2.join(", ") + "}" : n2[0];
      }
      elementName(e3, n2, s2) {
        return s2 === t2.EOF ? "<EOF>" : s2 === t2.EPSILON ? "<EPSILON>" : e3[s2] || n2[s2];
      }
      get length() {
        return this.intervals.map((t3) => t3.length).reduce((t3, e3) => t3 + e3);
      }
    }
    class _2 {
      constructor() {
        this.atn = null, this.stateNumber = _2.INVALID_STATE_NUMBER, this.stateType = null, this.ruleIndex = 0, this.epsilonOnlyTransitions = false, this.transitions = [], this.nextTokenWithinRule = null;
      }
      toString() {
        return this.stateNumber;
      }
      equals(t3) {
        return t3 instanceof _2 && this.stateNumber === t3.stateNumber;
      }
      isNonGreedyExitState() {
        return false;
      }
      addTransition(t3, e3) {
        e3 === void 0 && (e3 = -1), this.transitions.length === 0 ? this.epsilonOnlyTransitions = t3.isEpsilon : this.epsilonOnlyTransitions !== t3.isEpsilon && (this.epsilonOnlyTransitions = false), e3 === -1 ? this.transitions.push(t3) : this.transitions.splice(e3, 1, t3);
      }
    }
    _2.INVALID_TYPE = 0, _2.BASIC = 1, _2.RULE_START = 2, _2.BLOCK_START = 3, _2.PLUS_BLOCK_START = 4, _2.STAR_BLOCK_START = 5, _2.TOKEN_START = 6, _2.RULE_STOP = 7, _2.BLOCK_END = 8, _2.STAR_LOOP_BACK = 9, _2.STAR_LOOP_ENTRY = 10, _2.PLUS_LOOP_BACK = 11, _2.LOOP_END = 12, _2.serializationNames = ["INVALID", "BASIC", "RULE_START", "BLOCK_START", "PLUS_BLOCK_START", "STAR_BLOCK_START", "TOKEN_START", "RULE_STOP", "BLOCK_END", "STAR_LOOP_BACK", "STAR_LOOP_ENTRY", "PLUS_LOOP_BACK", "LOOP_END"], _2.INVALID_STATE_NUMBER = -1;
    class A2 extends _2 {
      constructor() {
        return super(), this.stateType = _2.RULE_STOP, this;
      }
    }
    class C2 {
      constructor(t3) {
        if (t3 == null)
          throw "target cannot be null.";
        this.target = t3, this.isEpsilon = false, this.label = null;
      }
    }
    C2.EPSILON = 1, C2.RANGE = 2, C2.RULE = 3, C2.PREDICATE = 4, C2.ATOM = 5, C2.ACTION = 6, C2.SET = 7, C2.NOT_SET = 8, C2.WILDCARD = 9, C2.PRECEDENCE = 10, C2.serializationNames = ["INVALID", "EPSILON", "RANGE", "RULE", "PREDICATE", "ATOM", "ACTION", "SET", "NOT_SET", "WILDCARD", "PRECEDENCE"], C2.serializationTypes = {EpsilonTransition: C2.EPSILON, RangeTransition: C2.RANGE, RuleTransition: C2.RULE, PredicateTransition: C2.PREDICATE, AtomTransition: C2.ATOM, ActionTransition: C2.ACTION, SetTransition: C2.SET, NotSetTransition: C2.NOT_SET, WildcardTransition: C2.WILDCARD, PrecedencePredicateTransition: C2.PRECEDENCE};
    class N2 extends C2 {
      constructor(t3, e3, n2, s2) {
        super(t3), this.ruleIndex = e3, this.precedence = n2, this.followState = s2, this.serializationType = C2.RULE, this.isEpsilon = true;
      }
      matches(t3, e3, n2) {
        return false;
      }
    }
    class y2 extends C2 {
      constructor(e3, n2) {
        super(e3), this.serializationType = C2.SET, n2 != null ? this.label = n2 : (this.label = new E2(), this.label.addOne(t2.INVALID_TYPE));
      }
      matches(t3, e3, n2) {
        return this.label.contains(t3);
      }
      toString() {
        return this.label.toString();
      }
    }
    class I2 extends y2 {
      constructor(t3, e3) {
        super(t3, e3), this.serializationType = C2.NOT_SET;
      }
      matches(t3, e3, n2) {
        return t3 >= e3 && t3 <= n2 && !super.matches(t3, e3, n2);
      }
      toString() {
        return "~" + super.toString();
      }
    }
    class k2 extends C2 {
      constructor(t3) {
        super(t3), this.serializationType = C2.WILDCARD;
      }
      matches(t3, e3, n2) {
        return t3 >= e3 && t3 <= n2;
      }
      toString() {
        return ".";
      }
    }
    class L2 extends C2 {
      constructor(t3) {
        super(t3);
      }
    }
    class O2 {
    }
    class v2 extends O2 {
    }
    class R2 extends v2 {
    }
    class w2 extends R2 {
      get ruleContext() {
        throw new Error("missing interface implementation");
      }
    }
    class P2 extends R2 {
    }
    class b2 extends P2 {
    }
    const D2 = {toStringTree: function(t3, e3, n2) {
      e3 = e3 || null, (n2 = n2 || null) !== null && (e3 = n2.ruleNames);
      let s2 = D2.getNodeText(t3, e3);
      s2 = function(t4, e4) {
        return t4 = t4.replace(/\t/g, "\\t").replace(/\n/g, "\\n").replace(/\r/g, "\\r");
      }(s2);
      const i3 = t3.getChildCount();
      if (i3 === 0)
        return s2;
      let r3 = "(" + s2 + " ";
      i3 > 0 && (s2 = D2.toStringTree(t3.getChild(0), e3), r3 = r3.concat(s2));
      for (let n3 = 1; n3 < i3; n3++)
        s2 = D2.toStringTree(t3.getChild(n3), e3), r3 = r3.concat(" " + s2);
      return r3 = r3.concat(")"), r3;
    }, getNodeText: function(e3, n2, s2) {
      if (n2 = n2 || null, (s2 = s2 || null) !== null && (n2 = s2.ruleNames), n2 !== null) {
        if (e3 instanceof w2) {
          const t3 = e3.ruleContext.getAltNumber();
          return t3 != 0 ? n2[e3.ruleIndex] + ":" + t3 : n2[e3.ruleIndex];
        }
        if (e3 instanceof b2)
          return e3.toString();
        if (e3 instanceof P2 && e3.symbol !== null)
          return e3.symbol.text;
      }
      const i3 = e3.getPayload();
      return i3 instanceof t2 ? i3.text : e3.getPayload().toString();
    }, getChildren: function(t3) {
      const e3 = [];
      for (let n2 = 0; n2 < t3.getChildCount(); n2++)
        e3.push(t3.getChild(n2));
      return e3;
    }, getAncestors: function(t3) {
      let e3 = [];
      for (t3 = t3.getParent(); t3 !== null; )
        e3 = [t3].concat(e3), t3 = t3.getParent();
      return e3;
    }, findAllTokenNodes: function(t3, e3) {
      return D2.findAllNodes(t3, e3, true);
    }, findAllRuleNodes: function(t3, e3) {
      return D2.findAllNodes(t3, e3, false);
    }, findAllNodes: function(t3, e3, n2) {
      const s2 = [];
      return D2._findAllNodes(t3, e3, n2, s2), s2;
    }, _findAllNodes: function(t3, e3, n2, s2) {
      n2 && t3 instanceof P2 ? t3.symbol.type === e3 && s2.push(t3) : !n2 && t3 instanceof w2 && t3.ruleIndex === e3 && s2.push(t3);
      for (let i3 = 0; i3 < t3.getChildCount(); i3++)
        D2._findAllNodes(t3.getChild(i3), e3, n2, s2);
    }, descendants: function(t3) {
      let e3 = [t3];
      for (let n2 = 0; n2 < t3.getChildCount(); n2++)
        e3 = e3.concat(D2.descendants(t3.getChild(n2)));
      return e3;
    }}, F2 = D2;
    class M2 extends w2 {
      constructor(t3, e3) {
        super(), this.parentCtx = t3 || null, this.invokingState = e3 || -1;
      }
      depth() {
        let t3 = 0, e3 = this;
        for (; e3 !== null; )
          e3 = e3.parentCtx, t3 += 1;
        return t3;
      }
      isEmpty() {
        return this.invokingState === -1;
      }
      getSourceInterval() {
        return m2.INVALID_INTERVAL;
      }
      get ruleContext() {
        return this;
      }
      getPayload() {
        return this;
      }
      getText() {
        return this.getChildCount() === 0 ? "" : this.children.map(function(t3) {
          return t3.getText();
        }).join("");
      }
      getAltNumber() {
        return 0;
      }
      setAltNumber(t3) {
      }
      getChild(t3) {
        return null;
      }
      getChildCount() {
        return 0;
      }
      accept(t3) {
        return t3.visitChildren(this);
      }
      toStringTree(t3, e3) {
        return F2.toStringTree(this, t3, e3);
      }
      toString(t3, e3) {
        t3 = t3 || null, e3 = e3 || null;
        let n2 = this, s2 = "[";
        for (; n2 !== null && n2 !== e3; ) {
          if (t3 === null)
            n2.isEmpty() || (s2 += n2.invokingState);
          else {
            const e4 = n2.ruleIndex;
            s2 += e4 >= 0 && e4 < t3.length ? t3[e4] : "" + e4;
          }
          n2.parentCtx === null || t3 === null && n2.parentCtx.isEmpty() || (s2 += " "), n2 = n2.parentCtx;
        }
        return s2 += "]", s2;
      }
    }
    class U {
      constructor(t3) {
        this.cachedHashCode = t3;
      }
      isEmpty() {
        return this === U.EMPTY;
      }
      hasEmptyPath() {
        return this.getReturnState(this.length - 1) === U.EMPTY_RETURN_STATE;
      }
      hashCode() {
        return this.cachedHashCode;
      }
      updateHashCode(t3) {
        t3.update(this.cachedHashCode);
      }
    }
    U.EMPTY = null, U.EMPTY_RETURN_STATE = 2147483647, U.globalNodeCount = 1, U.id = U.globalNodeCount, U.trace_atn_sim = false;
    class B extends U {
      constructor(t3, e3) {
        const n2 = new o2();
        return n2.update(t3, e3), super(n2.finish()), this.parents = t3, this.returnStates = e3, this;
      }
      isEmpty() {
        return this.returnStates[0] === U.EMPTY_RETURN_STATE;
      }
      getParent(t3) {
        return this.parents[t3];
      }
      getReturnState(t3) {
        return this.returnStates[t3];
      }
      equals(t3) {
        return this === t3 || t3 instanceof B && this.hashCode() === t3.hashCode() && e2(this.returnStates, t3.returnStates) && e2(this.parents, t3.parents);
      }
      toString() {
        if (this.isEmpty())
          return "[]";
        {
          let t3 = "[";
          for (let e3 = 0; e3 < this.returnStates.length; e3++)
            e3 > 0 && (t3 += ", "), this.returnStates[e3] !== U.EMPTY_RETURN_STATE ? (t3 += this.returnStates[e3], this.parents[e3] !== null ? t3 = t3 + " " + this.parents[e3] : t3 += "null") : t3 += "$";
          return t3 + "]";
        }
      }
      get length() {
        return this.returnStates.length;
      }
    }
    class V extends U {
      constructor(t3, e3) {
        let n2 = 0;
        const s2 = new o2();
        t3 !== null ? s2.update(t3, e3) : s2.update(1), n2 = s2.finish(), super(n2), this.parentCtx = t3, this.returnState = e3;
      }
      getParent(t3) {
        return this.parentCtx;
      }
      getReturnState(t3) {
        return this.returnState;
      }
      equals(t3) {
        return this === t3 || t3 instanceof V && this.hashCode() === t3.hashCode() && this.returnState === t3.returnState && (this.parentCtx == null ? t3.parentCtx == null : this.parentCtx.equals(t3.parentCtx));
      }
      toString() {
        const t3 = this.parentCtx === null ? "" : this.parentCtx.toString();
        return t3.length === 0 ? this.returnState === U.EMPTY_RETURN_STATE ? "$" : "" + this.returnState : this.returnState + " " + t3;
      }
      get length() {
        return 1;
      }
      static create(t3, e3) {
        return e3 === U.EMPTY_RETURN_STATE && t3 === null ? U.EMPTY : new V(t3, e3);
      }
    }
    class z extends V {
      constructor() {
        super(null, U.EMPTY_RETURN_STATE);
      }
      isEmpty() {
        return true;
      }
      getParent(t3) {
        return null;
      }
      getReturnState(t3) {
        return this.returnState;
      }
      equals(t3) {
        return this === t3;
      }
      toString() {
        return "$";
      }
    }
    U.EMPTY = new z();
    const q = "h-";
    class H {
      constructor(t3, e3) {
        this.data = {}, this.hashFunction = t3 || a2, this.equalsFunction = e3 || l2;
      }
      set(t3, e3) {
        const n2 = q + this.hashFunction(t3);
        if (n2 in this.data) {
          const s2 = this.data[n2];
          for (let n3 = 0; n3 < s2.length; n3++) {
            const i3 = s2[n3];
            if (this.equalsFunction(t3, i3.key)) {
              const t4 = i3.value;
              return i3.value = e3, t4;
            }
          }
          return s2.push({key: t3, value: e3}), e3;
        }
        return this.data[n2] = [{key: t3, value: e3}], e3;
      }
      containsKey(t3) {
        const e3 = q + this.hashFunction(t3);
        if (e3 in this.data) {
          const n2 = this.data[e3];
          for (let e4 = 0; e4 < n2.length; e4++) {
            const s2 = n2[e4];
            if (this.equalsFunction(t3, s2.key))
              return true;
          }
        }
        return false;
      }
      get(t3) {
        const e3 = q + this.hashFunction(t3);
        if (e3 in this.data) {
          const n2 = this.data[e3];
          for (let e4 = 0; e4 < n2.length; e4++) {
            const s2 = n2[e4];
            if (this.equalsFunction(t3, s2.key))
              return s2.value;
          }
        }
        return null;
      }
      entries() {
        return Object.keys(this.data).filter((t3) => t3.startsWith(q)).flatMap((t3) => this.data[t3], this);
      }
      getKeys() {
        return this.entries().map((t3) => t3.key);
      }
      getValues() {
        return this.entries().map((t3) => t3.value);
      }
      toString() {
        return "[" + this.entries().map((t3) => "{" + t3.key + ":" + t3.value + "}").join(", ") + "]";
      }
      get length() {
        return Object.keys(this.data).filter((t3) => t3.startsWith(q)).map((t3) => this.data[t3].length, this).reduce((t3, e3) => t3 + e3, 0);
      }
    }
    function K(t3, e3) {
      if (e3 == null && (e3 = M2.EMPTY), e3.parentCtx === null || e3 === M2.EMPTY)
        return U.EMPTY;
      const n2 = K(t3, e3.parentCtx), s2 = t3.states[e3.invokingState].transitions[0];
      return V.create(n2, s2.followState.stateNumber);
    }
    function Y(t3, e3, n2) {
      if (t3.isEmpty())
        return t3;
      let s2 = n2.get(t3) || null;
      if (s2 !== null)
        return s2;
      if (s2 = e3.get(t3), s2 !== null)
        return n2.set(t3, s2), s2;
      let i3 = false, r3 = [];
      for (let s3 = 0; s3 < r3.length; s3++) {
        const o4 = Y(t3.getParent(s3), e3, n2);
        if (i3 || o4 !== t3.getParent(s3)) {
          if (!i3) {
            r3 = [];
            for (let e4 = 0; e4 < t3.length; e4++)
              r3[e4] = t3.getParent(e4);
            i3 = true;
          }
          r3[s3] = o4;
        }
      }
      if (!i3)
        return e3.add(t3), n2.set(t3, t3), t3;
      let o3 = null;
      return o3 = r3.length === 0 ? U.EMPTY : r3.length === 1 ? V.create(r3[0], t3.getReturnState(0)) : new B(r3, t3.returnStates), e3.add(o3), n2.set(o3, o3), n2.set(t3, o3), o3;
    }
    function G(t3, e3, n2, s2) {
      if (t3 === e3)
        return t3;
      if (t3 instanceof V && e3 instanceof V)
        return function(t4, e4, n3, s3) {
          if (s3 !== null) {
            let n4 = s3.get(t4, e4);
            if (n4 !== null)
              return n4;
            if (n4 = s3.get(e4, t4), n4 !== null)
              return n4;
          }
          const i3 = function(t5, e5, n4) {
            if (n4) {
              if (t5 === U.EMPTY)
                return U.EMPTY;
              if (e5 === U.EMPTY)
                return U.EMPTY;
            } else {
              if (t5 === U.EMPTY && e5 === U.EMPTY)
                return U.EMPTY;
              if (t5 === U.EMPTY) {
                const t6 = [e5.returnState, U.EMPTY_RETURN_STATE], n5 = [e5.parentCtx, null];
                return new B(n5, t6);
              }
              if (e5 === U.EMPTY) {
                const e6 = [t5.returnState, U.EMPTY_RETURN_STATE], n5 = [t5.parentCtx, null];
                return new B(n5, e6);
              }
            }
            return null;
          }(t4, e4, n3);
          if (i3 !== null)
            return s3 !== null && s3.set(t4, e4, i3), i3;
          if (t4.returnState === e4.returnState) {
            const i4 = G(t4.parentCtx, e4.parentCtx, n3, s3);
            if (i4 === t4.parentCtx)
              return t4;
            if (i4 === e4.parentCtx)
              return e4;
            const r3 = V.create(i4, t4.returnState);
            return s3 !== null && s3.set(t4, e4, r3), r3;
          }
          {
            let n4 = null;
            if ((t4 === e4 || t4.parentCtx !== null && t4.parentCtx === e4.parentCtx) && (n4 = t4.parentCtx), n4 !== null) {
              const i5 = [t4.returnState, e4.returnState];
              t4.returnState > e4.returnState && (i5[0] = e4.returnState, i5[1] = t4.returnState);
              const r4 = new B([n4, n4], i5);
              return s3 !== null && s3.set(t4, e4, r4), r4;
            }
            const i4 = [t4.returnState, e4.returnState];
            let r3 = [t4.parentCtx, e4.parentCtx];
            t4.returnState > e4.returnState && (i4[0] = e4.returnState, i4[1] = t4.returnState, r3 = [e4.parentCtx, t4.parentCtx]);
            const o3 = new B(r3, i4);
            return s3 !== null && s3.set(t4, e4, o3), o3;
          }
        }(t3, e3, n2, s2);
      if (n2) {
        if (t3 instanceof z)
          return t3;
        if (e3 instanceof z)
          return e3;
      }
      return t3 instanceof V && (t3 = new B([t3.getParent()], [t3.returnState])), e3 instanceof V && (e3 = new B([e3.getParent()], [e3.returnState])), function(t4, e4, n3, s3) {
        if (s3 !== null) {
          let n4 = s3.get(t4, e4);
          if (n4 !== null)
            return U.trace_atn_sim && console.log("mergeArrays a=" + t4 + ",b=" + e4 + " -> previous"), n4;
          if (n4 = s3.get(e4, t4), n4 !== null)
            return U.trace_atn_sim && console.log("mergeArrays a=" + t4 + ",b=" + e4 + " -> previous"), n4;
        }
        let i3 = 0, r3 = 0, o3 = 0, a3 = new Array(t4.returnStates.length + e4.returnStates.length).fill(0), l3 = new Array(t4.returnStates.length + e4.returnStates.length).fill(null);
        for (; i3 < t4.returnStates.length && r3 < e4.returnStates.length; ) {
          const h4 = t4.parents[i3], c3 = e4.parents[r3];
          if (t4.returnStates[i3] === e4.returnStates[r3]) {
            const e5 = t4.returnStates[i3];
            e5 === U.EMPTY_RETURN_STATE && h4 === null && c3 === null || h4 !== null && c3 !== null && h4 === c3 ? (l3[o3] = h4, a3[o3] = e5) : (l3[o3] = G(h4, c3, n3, s3), a3[o3] = e5), i3 += 1, r3 += 1;
          } else
            t4.returnStates[i3] < e4.returnStates[r3] ? (l3[o3] = h4, a3[o3] = t4.returnStates[i3], i3 += 1) : (l3[o3] = c3, a3[o3] = e4.returnStates[r3], r3 += 1);
          o3 += 1;
        }
        if (i3 < t4.returnStates.length)
          for (let e5 = i3; e5 < t4.returnStates.length; e5++)
            l3[o3] = t4.parents[e5], a3[o3] = t4.returnStates[e5], o3 += 1;
        else
          for (let t5 = r3; t5 < e4.returnStates.length; t5++)
            l3[o3] = e4.parents[t5], a3[o3] = e4.returnStates[t5], o3 += 1;
        if (o3 < l3.length) {
          if (o3 === 1) {
            const n4 = V.create(l3[0], a3[0]);
            return s3 !== null && s3.set(t4, e4, n4), n4;
          }
          l3 = l3.slice(0, o3), a3 = a3.slice(0, o3);
        }
        const h3 = new B(l3, a3);
        return h3.equals(t4) ? (s3 !== null && s3.set(t4, e4, t4), U.trace_atn_sim && console.log("mergeArrays a=" + t4 + ",b=" + e4 + " -> a"), t4) : h3.equals(e4) ? (s3 !== null && s3.set(t4, e4, e4), U.trace_atn_sim && console.log("mergeArrays a=" + t4 + ",b=" + e4 + " -> b"), e4) : (function(t5) {
          const e5 = new H();
          for (let n4 = 0; n4 < t5.length; n4++) {
            const s4 = t5[n4];
            e5.containsKey(s4) || e5.set(s4, s4);
          }
          for (let n4 = 0; n4 < t5.length; n4++)
            t5[n4] = e5.get(t5[n4]);
        }(l3), s3 !== null && s3.set(t4, e4, h3), U.trace_atn_sim && console.log("mergeArrays a=" + t4 + ",b=" + e4 + " -> " + h3), h3);
      }(t3, e3, n2, s2);
    }
    class j {
      constructor() {
        this.data = [];
      }
      add(t3) {
        this.data[t3] = true;
      }
      or(t3) {
        Object.keys(t3.data).map((t4) => this.add(t4), this);
      }
      remove(t3) {
        delete this.data[t3];
      }
      has(t3) {
        return this.data[t3] === true;
      }
      values() {
        return Object.keys(this.data);
      }
      minValue() {
        return Math.min.apply(null, this.values());
      }
      hashCode() {
        return o2.hashStuff(this.values());
      }
      equals(t3) {
        return t3 instanceof j && e2(this.data, t3.data);
      }
      toString() {
        return "{" + this.values().join(", ") + "}";
      }
      get length() {
        return this.values().length;
      }
    }
    class W {
      constructor(t3) {
        this.atn = t3;
      }
      getDecisionLookahead(t3) {
        if (t3 === null)
          return null;
        const e3 = t3.transitions.length, n2 = [];
        for (let s2 = 0; s2 < e3; s2++) {
          n2[s2] = new E2();
          const e4 = new d2(), i3 = false;
          this._LOOK(t3.transition(s2).target, null, U.EMPTY, n2[s2], e4, new j(), i3, false), (n2[s2].length === 0 || n2[s2].contains(W.HIT_PRED)) && (n2[s2] = null);
        }
        return n2;
      }
      LOOK(t3, e3, n2) {
        const s2 = new E2(), i3 = (n2 = n2 || null) !== null ? K(t3.atn, n2) : null;
        return this._LOOK(t3, e3, i3, s2, new d2(), new j(), true, true), s2;
      }
      _LOOK(e3, n2, s2, i3, r3, o3, a3, l3) {
        const h3 = new S2({state: e3, alt: 0, context: s2}, null);
        if (!r3.has(h3)) {
          if (r3.add(h3), e3 === n2) {
            if (s2 === null)
              return void i3.addOne(t2.EPSILON);
            if (s2.isEmpty() && l3)
              return void i3.addOne(t2.EOF);
          }
          if (e3 instanceof A2) {
            if (s2 === null)
              return void i3.addOne(t2.EPSILON);
            if (s2.isEmpty() && l3)
              return void i3.addOne(t2.EOF);
            if (s2 !== U.EMPTY) {
              const t3 = o3.has(e3.ruleIndex);
              try {
                o3.remove(e3.ruleIndex);
                for (let t4 = 0; t4 < s2.length; t4++) {
                  const e4 = this.atn.states[s2.getReturnState(t4)];
                  this._LOOK(e4, n2, s2.getParent(t4), i3, r3, o3, a3, l3);
                }
              } finally {
                t3 && o3.add(e3.ruleIndex);
              }
              return;
            }
          }
          for (let h4 = 0; h4 < e3.transitions.length; h4++) {
            const c3 = e3.transitions[h4];
            if (c3.constructor === N2) {
              if (o3.has(c3.target.ruleIndex))
                continue;
              const t3 = V.create(s2, c3.followState.stateNumber);
              try {
                o3.add(c3.target.ruleIndex), this._LOOK(c3.target, n2, t3, i3, r3, o3, a3, l3);
              } finally {
                o3.remove(c3.target.ruleIndex);
              }
            } else if (c3 instanceof L2)
              a3 ? this._LOOK(c3.target, n2, s2, i3, r3, o3, a3, l3) : i3.addOne(W.HIT_PRED);
            else if (c3.isEpsilon)
              this._LOOK(c3.target, n2, s2, i3, r3, o3, a3, l3);
            else if (c3.constructor === k2)
              i3.addRange(t2.MIN_USER_TOKEN_TYPE, this.atn.maxTokenType);
            else {
              let e4 = c3.label;
              e4 !== null && (c3 instanceof I2 && (e4 = e4.complement(t2.MIN_USER_TOKEN_TYPE, this.atn.maxTokenType)), i3.addSet(e4));
            }
          }
        }
      }
    }
    W.HIT_PRED = t2.INVALID_TYPE;
    class $ {
      constructor(t3, e3) {
        this.grammarType = t3, this.maxTokenType = e3, this.states = [], this.decisionToState = [], this.ruleToStartState = [], this.ruleToStopState = null, this.modeNameToStartState = {}, this.ruleToTokenType = null, this.lexerActions = null, this.modeToStartState = [];
      }
      nextTokensInContext(t3, e3) {
        return new W(this).LOOK(t3, null, e3);
      }
      nextTokensNoContext(t3) {
        return t3.nextTokenWithinRule !== null || (t3.nextTokenWithinRule = this.nextTokensInContext(t3, null), t3.nextTokenWithinRule.readOnly = true), t3.nextTokenWithinRule;
      }
      nextTokens(t3, e3) {
        return e3 === void 0 ? this.nextTokensNoContext(t3) : this.nextTokensInContext(t3, e3);
      }
      addState(t3) {
        t3 !== null && (t3.atn = this, t3.stateNumber = this.states.length), this.states.push(t3);
      }
      removeState(t3) {
        this.states[t3.stateNumber] = null;
      }
      defineDecisionState(t3) {
        return this.decisionToState.push(t3), t3.decision = this.decisionToState.length - 1, t3.decision;
      }
      getDecisionState(t3) {
        return this.decisionToState.length === 0 ? null : this.decisionToState[t3];
      }
      getExpectedTokens(e3, n2) {
        if (e3 < 0 || e3 >= this.states.length)
          throw "Invalid state number.";
        const s2 = this.states[e3];
        let i3 = this.nextTokens(s2);
        if (!i3.contains(t2.EPSILON))
          return i3;
        const r3 = new E2();
        for (r3.addSet(i3), r3.removeOne(t2.EPSILON); n2 !== null && n2.invokingState >= 0 && i3.contains(t2.EPSILON); ) {
          const e4 = this.states[n2.invokingState].transitions[0];
          i3 = this.nextTokens(e4.followState), r3.addSet(i3), r3.removeOne(t2.EPSILON), n2 = n2.parentCtx;
        }
        return i3.contains(t2.EPSILON) && r3.addOne(t2.EOF), r3;
      }
    }
    $.INVALID_ALT_NUMBER = 0;
    class X extends _2 {
      constructor() {
        super(), this.stateType = _2.BASIC;
      }
    }
    class J extends _2 {
      constructor() {
        return super(), this.decision = -1, this.nonGreedy = false, this;
      }
    }
    class Z extends J {
      constructor() {
        return super(), this.endState = null, this;
      }
    }
    class Q extends _2 {
      constructor() {
        return super(), this.stateType = _2.BLOCK_END, this.startState = null, this;
      }
    }
    class tt extends _2 {
      constructor() {
        return super(), this.stateType = _2.LOOP_END, this.loopBackState = null, this;
      }
    }
    class et extends _2 {
      constructor() {
        return super(), this.stateType = _2.RULE_START, this.stopState = null, this.isPrecedenceRule = false, this;
      }
    }
    class nt extends J {
      constructor() {
        return super(), this.stateType = _2.TOKEN_START, this;
      }
    }
    class st extends J {
      constructor() {
        return super(), this.stateType = _2.PLUS_LOOP_BACK, this;
      }
    }
    class it extends _2 {
      constructor() {
        return super(), this.stateType = _2.STAR_LOOP_BACK, this;
      }
    }
    class rt extends J {
      constructor() {
        return super(), this.stateType = _2.STAR_LOOP_ENTRY, this.loopBackState = null, this.isPrecedenceDecision = null, this;
      }
    }
    class ot extends Z {
      constructor() {
        return super(), this.stateType = _2.PLUS_BLOCK_START, this.loopBackState = null, this;
      }
    }
    class at extends Z {
      constructor() {
        return super(), this.stateType = _2.STAR_BLOCK_START, this;
      }
    }
    class lt extends Z {
      constructor() {
        return super(), this.stateType = _2.BLOCK_START, this;
      }
    }
    class ht extends C2 {
      constructor(t3, e3) {
        super(t3), this.label_ = e3, this.label = this.makeLabel(), this.serializationType = C2.ATOM;
      }
      makeLabel() {
        const t3 = new E2();
        return t3.addOne(this.label_), t3;
      }
      matches(t3, e3, n2) {
        return this.label_ === t3;
      }
      toString() {
        return this.label_;
      }
    }
    class ct extends C2 {
      constructor(t3, e3, n2) {
        super(t3), this.serializationType = C2.RANGE, this.start = e3, this.stop = n2, this.label = this.makeLabel();
      }
      makeLabel() {
        const t3 = new E2();
        return t3.addRange(this.start, this.stop), t3;
      }
      matches(t3, e3, n2) {
        return t3 >= this.start && t3 <= this.stop;
      }
      toString() {
        return "'" + String.fromCharCode(this.start) + "'..'" + String.fromCharCode(this.stop) + "'";
      }
    }
    class ut extends C2 {
      constructor(t3, e3, n2, s2) {
        super(t3), this.serializationType = C2.ACTION, this.ruleIndex = e3, this.actionIndex = n2 === void 0 ? -1 : n2, this.isCtxDependent = s2 !== void 0 && s2, this.isEpsilon = true;
      }
      matches(t3, e3, n2) {
        return false;
      }
      toString() {
        return "action_" + this.ruleIndex + ":" + this.actionIndex;
      }
    }
    class dt extends C2 {
      constructor(t3, e3) {
        super(t3), this.serializationType = C2.EPSILON, this.isEpsilon = true, this.outermostPrecedenceReturn = e3;
      }
      matches(t3, e3, n2) {
        return false;
      }
      toString() {
        return "epsilon";
      }
    }
    class pt extends p2 {
      constructor(t3, e3, n2) {
        super(), this.ruleIndex = t3 === void 0 ? -1 : t3, this.predIndex = e3 === void 0 ? -1 : e3, this.isCtxDependent = n2 !== void 0 && n2;
      }
      evaluate(t3, e3) {
        const n2 = this.isCtxDependent ? e3 : null;
        return t3.sempred(n2, this.ruleIndex, this.predIndex);
      }
      updateHashCode(t3) {
        t3.update(this.ruleIndex, this.predIndex, this.isCtxDependent);
      }
      equals(t3) {
        return this === t3 || t3 instanceof pt && this.ruleIndex === t3.ruleIndex && this.predIndex === t3.predIndex && this.isCtxDependent === t3.isCtxDependent;
      }
      toString() {
        return "{" + this.ruleIndex + ":" + this.predIndex + "}?";
      }
    }
    p2.NONE = new pt();
    class gt extends L2 {
      constructor(t3, e3, n2, s2) {
        super(t3), this.serializationType = C2.PREDICATE, this.ruleIndex = e3, this.predIndex = n2, this.isCtxDependent = s2, this.isEpsilon = true;
      }
      matches(t3, e3, n2) {
        return false;
      }
      getPredicate() {
        return new pt(this.ruleIndex, this.predIndex, this.isCtxDependent);
      }
      toString() {
        return "pred_" + this.ruleIndex + ":" + this.predIndex;
      }
    }
    class ft extends p2 {
      constructor(t3) {
        super(), this.precedence = t3 === void 0 ? 0 : t3;
      }
      evaluate(t3, e3) {
        return t3.precpred(e3, this.precedence);
      }
      evalPrecedence(t3, e3) {
        return t3.precpred(e3, this.precedence) ? p2.NONE : null;
      }
      compareTo(t3) {
        return this.precedence - t3.precedence;
      }
      updateHashCode(t3) {
        t3.update(this.precedence);
      }
      equals(t3) {
        return this === t3 || t3 instanceof ft && this.precedence === t3.precedence;
      }
      toString() {
        return "{" + this.precedence + ">=prec}?";
      }
    }
    p2.PrecedencePredicate = ft;
    class xt extends L2 {
      constructor(t3, e3) {
        super(t3), this.serializationType = C2.PRECEDENCE, this.precedence = e3, this.isEpsilon = true;
      }
      matches(t3, e3, n2) {
        return false;
      }
      getPredicate() {
        return new ft(this.precedence);
      }
      toString() {
        return this.precedence + " >= _p";
      }
    }
    class Tt {
      constructor(t3) {
        t3 === void 0 && (t3 = null), this.readOnly = false, this.verifyATN = t3 === null || t3.verifyATN, this.generateRuleBypassTransitions = t3 !== null && t3.generateRuleBypassTransitions;
      }
    }
    Tt.defaultOptions = new Tt(), Tt.defaultOptions.readOnly = true;
    class St {
      constructor(t3) {
        this.actionType = t3, this.isPositionDependent = false;
      }
      hashCode() {
        const t3 = new o2();
        return this.updateHashCode(t3), t3.finish();
      }
      updateHashCode(t3) {
        t3.update(this.actionType);
      }
      equals(t3) {
        return this === t3;
      }
    }
    class mt extends St {
      constructor() {
        super(6);
      }
      execute(t3) {
        t3.skip();
      }
      toString() {
        return "skip";
      }
    }
    mt.INSTANCE = new mt();
    class Et extends St {
      constructor(t3) {
        super(0), this.channel = t3;
      }
      execute(t3) {
        t3._channel = this.channel;
      }
      updateHashCode(t3) {
        t3.update(this.actionType, this.channel);
      }
      equals(t3) {
        return this === t3 || t3 instanceof Et && this.channel === t3.channel;
      }
      toString() {
        return "channel(" + this.channel + ")";
      }
    }
    class _t extends St {
      constructor(t3, e3) {
        super(1), this.ruleIndex = t3, this.actionIndex = e3, this.isPositionDependent = true;
      }
      execute(t3) {
        t3.action(null, this.ruleIndex, this.actionIndex);
      }
      updateHashCode(t3) {
        t3.update(this.actionType, this.ruleIndex, this.actionIndex);
      }
      equals(t3) {
        return this === t3 || t3 instanceof _t && this.ruleIndex === t3.ruleIndex && this.actionIndex === t3.actionIndex;
      }
    }
    class At extends St {
      constructor() {
        super(3);
      }
      execute(t3) {
        t3.more();
      }
      toString() {
        return "more";
      }
    }
    At.INSTANCE = new At();
    class Ct extends St {
      constructor(t3) {
        super(7), this.type = t3;
      }
      execute(t3) {
        t3.type = this.type;
      }
      updateHashCode(t3) {
        t3.update(this.actionType, this.type);
      }
      equals(t3) {
        return this === t3 || t3 instanceof Ct && this.type === t3.type;
      }
      toString() {
        return "type(" + this.type + ")";
      }
    }
    class Nt extends St {
      constructor(t3) {
        super(5), this.mode = t3;
      }
      execute(t3) {
        t3.pushMode(this.mode);
      }
      updateHashCode(t3) {
        t3.update(this.actionType, this.mode);
      }
      equals(t3) {
        return this === t3 || t3 instanceof Nt && this.mode === t3.mode;
      }
      toString() {
        return "pushMode(" + this.mode + ")";
      }
    }
    class yt extends St {
      constructor() {
        super(4);
      }
      execute(t3) {
        t3.popMode();
      }
      toString() {
        return "popMode";
      }
    }
    yt.INSTANCE = new yt();
    class It extends St {
      constructor(t3) {
        super(2), this.mode = t3;
      }
      execute(t3) {
        t3.mode(this.mode);
      }
      updateHashCode(t3) {
        t3.update(this.actionType, this.mode);
      }
      equals(t3) {
        return this === t3 || t3 instanceof It && this.mode === t3.mode;
      }
      toString() {
        return "mode(" + this.mode + ")";
      }
    }
    function kt(t3, e3) {
      const n2 = [];
      return n2[t3 - 1] = e3, n2.map(function(t4) {
        return e3;
      });
    }
    class Lt {
      constructor(t3) {
        t3 == null && (t3 = Tt.defaultOptions), this.deserializationOptions = t3, this.stateFactories = null, this.actionFactories = null;
      }
      deserialize(t3) {
        const e3 = this.reset(t3);
        this.checkVersion(e3), e3 && this.skipUUID();
        const n2 = this.readATN();
        this.readStates(n2, e3), this.readRules(n2, e3), this.readModes(n2);
        const s2 = [];
        return this.readSets(n2, s2, this.readInt.bind(this)), e3 && this.readSets(n2, s2, this.readInt32.bind(this)), this.readEdges(n2, s2), this.readDecisions(n2), this.readLexerActions(n2, e3), this.markPrecedenceDecisions(n2), this.verifyATN(n2), this.deserializationOptions.generateRuleBypassTransitions && n2.grammarType === 1 && (this.generateRuleBypassTransitions(n2), this.verifyATN(n2)), n2;
      }
      reset(t3) {
        if ((t3.charCodeAt ? t3.charCodeAt(0) : t3[0]) === 3) {
          const e3 = function(t4) {
            const e4 = t4.charCodeAt(0);
            return e4 > 1 ? e4 - 2 : e4 + 65534;
          }, n2 = t3.split("").map(e3);
          return n2[0] = t3.charCodeAt(0), this.data = n2, this.pos = 0, true;
        }
        return this.data = t3, this.pos = 0, false;
      }
      skipUUID() {
        let t3 = 0;
        for (; t3++ < 8; )
          this.readInt();
      }
      checkVersion(t3) {
        const e3 = this.readInt();
        if (!t3 && e3 !== 4)
          throw "Could not deserialize ATN with version " + e3 + " (expected 4).";
      }
      readATN() {
        const t3 = this.readInt(), e3 = this.readInt();
        return new $(t3, e3);
      }
      readStates(t3, e3) {
        let n2, s2, i3;
        const r3 = [], o3 = [], a3 = this.readInt();
        for (let n3 = 0; n3 < a3; n3++) {
          const n4 = this.readInt();
          if (n4 === _2.INVALID_TYPE) {
            t3.addState(null);
            continue;
          }
          let s3 = this.readInt();
          e3 && s3 === 65535 && (s3 = -1);
          const i4 = this.stateFactory(n4, s3);
          if (n4 === _2.LOOP_END) {
            const t4 = this.readInt();
            r3.push([i4, t4]);
          } else if (i4 instanceof Z) {
            const t4 = this.readInt();
            o3.push([i4, t4]);
          }
          t3.addState(i4);
        }
        for (n2 = 0; n2 < r3.length; n2++)
          s2 = r3[n2], s2[0].loopBackState = t3.states[s2[1]];
        for (n2 = 0; n2 < o3.length; n2++)
          s2 = o3[n2], s2[0].endState = t3.states[s2[1]];
        let l3 = this.readInt();
        for (n2 = 0; n2 < l3; n2++)
          i3 = this.readInt(), t3.states[i3].nonGreedy = true;
        let h3 = this.readInt();
        for (n2 = 0; n2 < h3; n2++)
          i3 = this.readInt(), t3.states[i3].isPrecedenceRule = true;
      }
      readRules(e3, n2) {
        let s2;
        const i3 = this.readInt();
        for (e3.grammarType === 0 && (e3.ruleToTokenType = kt(i3, 0)), e3.ruleToStartState = kt(i3, 0), s2 = 0; s2 < i3; s2++) {
          const i4 = this.readInt();
          if (e3.ruleToStartState[s2] = e3.states[i4], e3.grammarType === 0) {
            let i5 = this.readInt();
            n2 && i5 === 65535 && (i5 = t2.EOF), e3.ruleToTokenType[s2] = i5;
          }
        }
        for (e3.ruleToStopState = kt(i3, 0), s2 = 0; s2 < e3.states.length; s2++) {
          const t3 = e3.states[s2];
          t3 instanceof A2 && (e3.ruleToStopState[t3.ruleIndex] = t3, e3.ruleToStartState[t3.ruleIndex].stopState = t3);
        }
      }
      readModes(t3) {
        const e3 = this.readInt();
        for (let n2 = 0; n2 < e3; n2++) {
          let e4 = this.readInt();
          t3.modeToStartState.push(t3.states[e4]);
        }
      }
      readSets(t3, e3, n2) {
        const s2 = this.readInt();
        for (let t4 = 0; t4 < s2; t4++) {
          const t5 = new E2();
          e3.push(t5);
          const s3 = this.readInt();
          this.readInt() !== 0 && t5.addOne(-1);
          for (let e4 = 0; e4 < s3; e4++) {
            const e5 = n2(), s4 = n2();
            t5.addRange(e5, s4);
          }
        }
      }
      readEdges(t3, e3) {
        let n2, s2, i3, r3, o3;
        const a3 = this.readInt();
        for (n2 = 0; n2 < a3; n2++) {
          const n3 = this.readInt(), s3 = this.readInt(), i4 = this.readInt(), o4 = this.readInt(), a4 = this.readInt(), l3 = this.readInt();
          r3 = this.edgeFactory(t3, i4, n3, s3, o4, a4, l3, e3), t3.states[n3].addTransition(r3);
        }
        for (n2 = 0; n2 < t3.states.length; n2++)
          for (i3 = t3.states[n2], s2 = 0; s2 < i3.transitions.length; s2++) {
            const e4 = i3.transitions[s2];
            if (!(e4 instanceof N2))
              continue;
            let n3 = -1;
            t3.ruleToStartState[e4.target.ruleIndex].isPrecedenceRule && e4.precedence === 0 && (n3 = e4.target.ruleIndex), r3 = new dt(e4.followState, n3), t3.ruleToStopState[e4.target.ruleIndex].addTransition(r3);
          }
        for (n2 = 0; n2 < t3.states.length; n2++) {
          if (i3 = t3.states[n2], i3 instanceof Z) {
            if (i3.endState === null)
              throw "IllegalState";
            if (i3.endState.startState !== null)
              throw "IllegalState";
            i3.endState.startState = i3;
          }
          if (i3 instanceof st)
            for (s2 = 0; s2 < i3.transitions.length; s2++)
              o3 = i3.transitions[s2].target, o3 instanceof ot && (o3.loopBackState = i3);
          else if (i3 instanceof it)
            for (s2 = 0; s2 < i3.transitions.length; s2++)
              o3 = i3.transitions[s2].target, o3 instanceof rt && (o3.loopBackState = i3);
        }
      }
      readDecisions(t3) {
        const e3 = this.readInt();
        for (let n2 = 0; n2 < e3; n2++) {
          const e4 = this.readInt(), s2 = t3.states[e4];
          t3.decisionToState.push(s2), s2.decision = n2;
        }
      }
      readLexerActions(t3, e3) {
        if (t3.grammarType === 0) {
          const n2 = this.readInt();
          t3.lexerActions = kt(n2, null);
          for (let s2 = 0; s2 < n2; s2++) {
            const n3 = this.readInt();
            let i3 = this.readInt();
            e3 && i3 === 65535 && (i3 = -1);
            let r3 = this.readInt();
            e3 && r3 === 65535 && (r3 = -1), t3.lexerActions[s2] = this.lexerActionFactory(n3, i3, r3);
          }
        }
      }
      generateRuleBypassTransitions(t3) {
        let e3;
        const n2 = t3.ruleToStartState.length;
        for (e3 = 0; e3 < n2; e3++)
          t3.ruleToTokenType[e3] = t3.maxTokenType + e3 + 1;
        for (e3 = 0; e3 < n2; e3++)
          this.generateRuleBypassTransition(t3, e3);
      }
      generateRuleBypassTransition(t3, e3) {
        let n2, s2;
        const i3 = new lt();
        i3.ruleIndex = e3, t3.addState(i3);
        const r3 = new Q();
        r3.ruleIndex = e3, t3.addState(r3), i3.endState = r3, t3.defineDecisionState(i3), r3.startState = i3;
        let o3 = null, a3 = null;
        if (t3.ruleToStartState[e3].isPrecedenceRule) {
          for (a3 = null, n2 = 0; n2 < t3.states.length; n2++)
            if (s2 = t3.states[n2], this.stateIsEndStateFor(s2, e3)) {
              a3 = s2, o3 = s2.loopBackState.transitions[0];
              break;
            }
          if (o3 === null)
            throw "Couldn't identify final state of the precedence rule prefix section.";
        } else
          a3 = t3.ruleToStopState[e3];
        for (n2 = 0; n2 < t3.states.length; n2++) {
          s2 = t3.states[n2];
          for (let t4 = 0; t4 < s2.transitions.length; t4++) {
            const e4 = s2.transitions[t4];
            e4 !== o3 && e4.target === a3 && (e4.target = r3);
          }
        }
        const l3 = t3.ruleToStartState[e3], h3 = l3.transitions.length;
        for (; h3 > 0; )
          i3.addTransition(l3.transitions[h3 - 1]), l3.transitions = l3.transitions.slice(-1);
        t3.ruleToStartState[e3].addTransition(new dt(i3)), r3.addTransition(new dt(a3));
        const c3 = new X();
        t3.addState(c3), c3.addTransition(new ht(r3, t3.ruleToTokenType[e3])), i3.addTransition(new dt(c3));
      }
      stateIsEndStateFor(t3, e3) {
        if (t3.ruleIndex !== e3)
          return null;
        if (!(t3 instanceof rt))
          return null;
        const n2 = t3.transitions[t3.transitions.length - 1].target;
        return n2 instanceof tt && n2.epsilonOnlyTransitions && n2.transitions[0].target instanceof A2 ? t3 : null;
      }
      markPrecedenceDecisions(t3) {
        for (let e3 = 0; e3 < t3.states.length; e3++) {
          const n2 = t3.states[e3];
          if (n2 instanceof rt && t3.ruleToStartState[n2.ruleIndex].isPrecedenceRule) {
            const t4 = n2.transitions[n2.transitions.length - 1].target;
            t4 instanceof tt && t4.epsilonOnlyTransitions && t4.transitions[0].target instanceof A2 && (n2.isPrecedenceDecision = true);
          }
        }
      }
      verifyATN(t3) {
        if (this.deserializationOptions.verifyATN)
          for (let e3 = 0; e3 < t3.states.length; e3++) {
            const n2 = t3.states[e3];
            if (n2 !== null)
              if (this.checkCondition(n2.epsilonOnlyTransitions || n2.transitions.length <= 1), n2 instanceof ot)
                this.checkCondition(n2.loopBackState !== null);
              else if (n2 instanceof rt)
                if (this.checkCondition(n2.loopBackState !== null), this.checkCondition(n2.transitions.length === 2), n2.transitions[0].target instanceof at)
                  this.checkCondition(n2.transitions[1].target instanceof tt), this.checkCondition(!n2.nonGreedy);
                else {
                  if (!(n2.transitions[0].target instanceof tt))
                    throw "IllegalState";
                  this.checkCondition(n2.transitions[1].target instanceof at), this.checkCondition(n2.nonGreedy);
                }
              else
                n2 instanceof it ? (this.checkCondition(n2.transitions.length === 1), this.checkCondition(n2.transitions[0].target instanceof rt)) : n2 instanceof tt ? this.checkCondition(n2.loopBackState !== null) : n2 instanceof et ? this.checkCondition(n2.stopState !== null) : n2 instanceof Z ? this.checkCondition(n2.endState !== null) : n2 instanceof Q ? this.checkCondition(n2.startState !== null) : n2 instanceof J ? this.checkCondition(n2.transitions.length <= 1 || n2.decision >= 0) : this.checkCondition(n2.transitions.length <= 1 || n2 instanceof A2);
          }
      }
      checkCondition(t3, e3) {
        if (!t3)
          throw e3 == null && (e3 = "IllegalState"), e3;
      }
      readInt() {
        return this.data[this.pos++];
      }
      readInt32() {
        return this.readInt() | this.readInt() << 16;
      }
      edgeFactory(e3, n2, s2, i3, r3, o3, a3, l3) {
        const h3 = e3.states[i3];
        switch (n2) {
          case C2.EPSILON:
            return new dt(h3);
          case C2.RANGE:
            return new ct(h3, a3 !== 0 ? t2.EOF : r3, o3);
          case C2.RULE:
            return new N2(e3.states[r3], o3, a3, h3);
          case C2.PREDICATE:
            return new gt(h3, r3, o3, a3 !== 0);
          case C2.PRECEDENCE:
            return new xt(h3, r3);
          case C2.ATOM:
            return new ht(h3, a3 !== 0 ? t2.EOF : r3);
          case C2.ACTION:
            return new ut(h3, r3, o3, a3 !== 0);
          case C2.SET:
            return new y2(h3, l3[r3]);
          case C2.NOT_SET:
            return new I2(h3, l3[r3]);
          case C2.WILDCARD:
            return new k2(h3);
          default:
            throw "The specified transition type: " + n2 + " is not valid.";
        }
      }
      stateFactory(t3, e3) {
        if (this.stateFactories === null) {
          const t4 = [];
          t4[_2.INVALID_TYPE] = null, t4[_2.BASIC] = () => new X(), t4[_2.RULE_START] = () => new et(), t4[_2.BLOCK_START] = () => new lt(), t4[_2.PLUS_BLOCK_START] = () => new ot(), t4[_2.STAR_BLOCK_START] = () => new at(), t4[_2.TOKEN_START] = () => new nt(), t4[_2.RULE_STOP] = () => new A2(), t4[_2.BLOCK_END] = () => new Q(), t4[_2.STAR_LOOP_BACK] = () => new it(), t4[_2.STAR_LOOP_ENTRY] = () => new rt(), t4[_2.PLUS_LOOP_BACK] = () => new st(), t4[_2.LOOP_END] = () => new tt(), this.stateFactories = t4;
        }
        if (t3 > this.stateFactories.length || this.stateFactories[t3] === null)
          throw "The specified state type " + t3 + " is not valid.";
        {
          const n2 = this.stateFactories[t3]();
          if (n2 !== null)
            return n2.ruleIndex = e3, n2;
        }
      }
      lexerActionFactory(t3, e3, n2) {
        if (this.actionFactories === null) {
          const t4 = [];
          t4[0] = (t5, e4) => new Et(t5), t4[1] = (t5, e4) => new _t(t5, e4), t4[2] = (t5, e4) => new It(t5), t4[3] = (t5, e4) => At.INSTANCE, t4[4] = (t5, e4) => yt.INSTANCE, t4[5] = (t5, e4) => new Nt(t5), t4[6] = (t5, e4) => mt.INSTANCE, t4[7] = (t5, e4) => new Ct(t5), this.actionFactories = t4;
        }
        if (t3 > this.actionFactories.length || this.actionFactories[t3] === null)
          throw "The specified lexer action type " + t3 + " is not valid.";
        return this.actionFactories[t3](e3, n2);
      }
    }
    class Ot {
      syntaxError(t3, e3, n2, s2, i3, r3) {
      }
      reportAmbiguity(t3, e3, n2, s2, i3, r3, o3) {
      }
      reportAttemptingFullContext(t3, e3, n2, s2, i3, r3) {
      }
      reportContextSensitivity(t3, e3, n2, s2, i3, r3) {
      }
    }
    class vt extends Ot {
      constructor() {
        super();
      }
      syntaxError(t3, e3, n2, s2, i3, r3) {
        console.error("line " + n2 + ":" + s2 + " " + i3);
      }
    }
    vt.INSTANCE = new vt();
    class Rt extends Ot {
      constructor(t3) {
        if (super(), t3 === null)
          throw "delegates";
        return this.delegates = t3, this;
      }
      syntaxError(t3, e3, n2, s2, i3, r3) {
        this.delegates.map((o3) => o3.syntaxError(t3, e3, n2, s2, i3, r3));
      }
      reportAmbiguity(t3, e3, n2, s2, i3, r3, o3) {
        this.delegates.map((a3) => a3.reportAmbiguity(t3, e3, n2, s2, i3, r3, o3));
      }
      reportAttemptingFullContext(t3, e3, n2, s2, i3, r3) {
        this.delegates.map((o3) => o3.reportAttemptingFullContext(t3, e3, n2, s2, i3, r3));
      }
      reportContextSensitivity(t3, e3, n2, s2, i3, r3) {
        this.delegates.map((o3) => o3.reportContextSensitivity(t3, e3, n2, s2, i3, r3));
      }
    }
    class wt {
      constructor() {
        this._listeners = [vt.INSTANCE], this._interp = null, this._stateNumber = -1;
      }
      checkVersion(t3) {
        const e3 = "4.13.1";
        e3 !== t3 && console.log("ANTLR runtime and generated code versions disagree: " + e3 + "!=" + t3);
      }
      addErrorListener(t3) {
        this._listeners.push(t3);
      }
      removeErrorListeners() {
        this._listeners = [];
      }
      getLiteralNames() {
        return Object.getPrototypeOf(this).constructor.literalNames || [];
      }
      getSymbolicNames() {
        return Object.getPrototypeOf(this).constructor.symbolicNames || [];
      }
      getTokenNames() {
        if (!this.tokenNames) {
          const t3 = this.getLiteralNames(), e3 = this.getSymbolicNames(), n2 = t3.length > e3.length ? t3.length : e3.length;
          this.tokenNames = [];
          for (let s2 = 0; s2 < n2; s2++)
            this.tokenNames[s2] = t3[s2] || e3[s2] || "<INVALID";
        }
        return this.tokenNames;
      }
      getTokenTypeMap() {
        const e3 = this.getTokenNames();
        if (e3 === null)
          throw "The current recognizer does not provide a list of token names.";
        let n2 = this.tokenTypeMapCache[e3];
        return n2 === void 0 && (n2 = e3.reduce(function(t3, e4, n3) {
          t3[e4] = n3;
        }), n2.EOF = t2.EOF, this.tokenTypeMapCache[e3] = n2), n2;
      }
      getRuleIndexMap() {
        const t3 = this.ruleNames;
        if (t3 === null)
          throw "The current recognizer does not provide a list of rule names.";
        let e3 = this.ruleIndexMapCache[t3];
        return e3 === void 0 && (e3 = t3.reduce(function(t4, e4, n2) {
          t4[e4] = n2;
        }), this.ruleIndexMapCache[t3] = e3), e3;
      }
      getTokenType(e3) {
        const n2 = this.getTokenTypeMap()[e3];
        return n2 !== void 0 ? n2 : t2.INVALID_TYPE;
      }
      getErrorHeader(t3) {
        return "line " + t3.getOffendingToken().line + ":" + t3.getOffendingToken().column;
      }
      getTokenErrorDisplay(e3) {
        if (e3 === null)
          return "<no token>";
        let n2 = e3.text;
        return n2 === null && (n2 = e3.type === t2.EOF ? "<EOF>" : "<" + e3.type + ">"), n2 = n2.replace("\n", "\\n").replace("\r", "\\r").replace("	", "\\t"), "'" + n2 + "'";
      }
      getErrorListenerDispatch() {
        return new Rt(this._listeners);
      }
      sempred(t3, e3, n2) {
        return true;
      }
      precpred(t3, e3) {
        return true;
      }
      get atn() {
        return this._interp.atn;
      }
      get state() {
        return this._stateNumber;
      }
      set state(t3) {
        this._stateNumber = t3;
      }
    }
    wt.tokenTypeMapCache = {}, wt.ruleIndexMapCache = {};
    class Pt extends t2 {
      constructor(e3, n2, s2, i3, r3) {
        super(), this.source = e3 !== void 0 ? e3 : Pt.EMPTY_SOURCE, this.type = n2 !== void 0 ? n2 : null, this.channel = s2 !== void 0 ? s2 : t2.DEFAULT_CHANNEL, this.start = i3 !== void 0 ? i3 : -1, this.stop = r3 !== void 0 ? r3 : -1, this.tokenIndex = -1, this.source[0] !== null ? (this.line = e3[0].line, this.column = e3[0].column) : this.column = -1;
      }
      clone() {
        const t3 = new Pt(this.source, this.type, this.channel, this.start, this.stop);
        return t3.tokenIndex = this.tokenIndex, t3.line = this.line, t3.column = this.column, t3.text = this.text, t3;
      }
      cloneWithType(e3) {
        const n2 = new Pt(this.source, e3, this.channel, this.start, this.stop);
        return n2.tokenIndex = this.tokenIndex, n2.line = this.line, n2.column = this.column, e3 === t2.EOF && (n2.text = ""), n2;
      }
      toString() {
        let t3 = this.text;
        return t3 = t3 !== null ? t3.replace(/\n/g, "\\n").replace(/\r/g, "\\r").replace(/\t/g, "\\t") : "<no text>", "[@" + this.tokenIndex + "," + this.start + ":" + this.stop + "='" + t3 + "',<" + this.type + ">" + (this.channel > 0 ? ",channel=" + this.channel : "") + "," + this.line + ":" + this.column + "]";
      }
      get text() {
        if (this._text !== null)
          return this._text;
        const t3 = this.getInputStream();
        if (t3 === null)
          return null;
        const e3 = t3.size;
        return this.start < e3 && this.stop < e3 ? t3.getText(this.start, this.stop) : "<EOF>";
      }
      set text(t3) {
        this._text = t3;
      }
    }
    Pt.EMPTY_SOURCE = [null, null];
    class bt {
    }
    class Dt extends bt {
      constructor(t3) {
        super(), this.copyText = t3 !== void 0 && t3;
      }
      create(t3, e3, n2, s2, i3, r3, o3, a3) {
        const l3 = new Pt(t3, e3, s2, i3, r3);
        return l3.line = o3, l3.column = a3, n2 !== null ? l3.text = n2 : this.copyText && t3[1] !== null && (l3.text = t3[1].getText(i3, r3)), l3;
      }
      createThin(t3, e3) {
        const n2 = new Pt(null, t3);
        return n2.text = e3, n2;
      }
    }
    Dt.DEFAULT = new Dt();
    class Ft extends Error {
      constructor(t3) {
        super(t3.message), Error.captureStackTrace && Error.captureStackTrace(this, Ft), this.message = t3.message, this.recognizer = t3.recognizer, this.input = t3.input, this.ctx = t3.ctx, this.offendingToken = null, this.offendingState = -1, this.recognizer !== null && (this.offendingState = this.recognizer.state);
      }
      getExpectedTokens() {
        return this.recognizer !== null ? this.recognizer.atn.getExpectedTokens(this.offendingState, this.ctx) : null;
      }
      toString() {
        return this.message;
      }
    }
    class Mt extends Ft {
      constructor(t3, e3, n2, s2) {
        super({message: "", recognizer: t3, input: e3, ctx: null}), this.startIndex = n2, this.deadEndConfigs = s2;
      }
      toString() {
        let t3 = "";
        return this.startIndex >= 0 && this.startIndex < this.input.size && (t3 = this.input.getText(new m2(this.startIndex, this.startIndex))), "LexerNoViableAltException" + t3;
      }
    }
    class Ut extends wt {
      constructor(e3) {
        super(), this._input = e3, this._factory = Dt.DEFAULT, this._tokenFactorySourcePair = [this, e3], this._interp = null, this._token = null, this._tokenStartCharIndex = -1, this._tokenStartLine = -1, this._tokenStartColumn = -1, this._hitEOF = false, this._channel = t2.DEFAULT_CHANNEL, this._type = t2.INVALID_TYPE, this._modeStack = [], this._mode = Ut.DEFAULT_MODE, this._text = null;
      }
      reset() {
        this._input !== null && this._input.seek(0), this._token = null, this._type = t2.INVALID_TYPE, this._channel = t2.DEFAULT_CHANNEL, this._tokenStartCharIndex = -1, this._tokenStartColumn = -1, this._tokenStartLine = -1, this._text = null, this._hitEOF = false, this._mode = Ut.DEFAULT_MODE, this._modeStack = [], this._interp.reset();
      }
      nextToken() {
        if (this._input === null)
          throw "nextToken requires a non-null input stream.";
        const e3 = this._input.mark();
        try {
          for (; ; ) {
            if (this._hitEOF)
              return this.emitEOF(), this._token;
            this._token = null, this._channel = t2.DEFAULT_CHANNEL, this._tokenStartCharIndex = this._input.index, this._tokenStartColumn = this._interp.column, this._tokenStartLine = this._interp.line, this._text = null;
            let e4 = false;
            for (; ; ) {
              this._type = t2.INVALID_TYPE;
              let n2 = Ut.SKIP;
              try {
                n2 = this._interp.match(this._input, this._mode);
              } catch (t3) {
                if (!(t3 instanceof Ft))
                  throw console.log(t3.stack), t3;
                this.notifyListeners(t3), this.recover(t3);
              }
              if (this._input.LA(1) === t2.EOF && (this._hitEOF = true), this._type === t2.INVALID_TYPE && (this._type = n2), this._type === Ut.SKIP) {
                e4 = true;
                break;
              }
              if (this._type !== Ut.MORE)
                break;
            }
            if (!e4)
              return this._token === null && this.emit(), this._token;
          }
        } finally {
          this._input.release(e3);
        }
      }
      skip() {
        this._type = Ut.SKIP;
      }
      more() {
        this._type = Ut.MORE;
      }
      mode(t3) {
        this._mode = t3;
      }
      pushMode(t3) {
        this._interp.debug && console.log("pushMode " + t3), this._modeStack.push(this._mode), this.mode(t3);
      }
      popMode() {
        if (this._modeStack.length === 0)
          throw "Empty Stack";
        return this._interp.debug && console.log("popMode back to " + this._modeStack.slice(0, -1)), this.mode(this._modeStack.pop()), this._mode;
      }
      emitToken(t3) {
        this._token = t3;
      }
      emit() {
        const t3 = this._factory.create(this._tokenFactorySourcePair, this._type, this._text, this._channel, this._tokenStartCharIndex, this.getCharIndex() - 1, this._tokenStartLine, this._tokenStartColumn);
        return this.emitToken(t3), t3;
      }
      emitEOF() {
        const e3 = this.column, n2 = this.line, s2 = this._factory.create(this._tokenFactorySourcePair, t2.EOF, null, t2.DEFAULT_CHANNEL, this._input.index, this._input.index - 1, n2, e3);
        return this.emitToken(s2), s2;
      }
      getCharIndex() {
        return this._input.index;
      }
      getAllTokens() {
        const e3 = [];
        let n2 = this.nextToken();
        for (; n2.type !== t2.EOF; )
          e3.push(n2), n2 = this.nextToken();
        return e3;
      }
      notifyListeners(t3) {
        const e3 = this._tokenStartCharIndex, n2 = this._input.index, s2 = this._input.getText(e3, n2), i3 = "token recognition error at: '" + this.getErrorDisplay(s2) + "'";
        this.getErrorListenerDispatch().syntaxError(this, null, this._tokenStartLine, this._tokenStartColumn, i3, t3);
      }
      getErrorDisplay(t3) {
        const e3 = [];
        for (let n2 = 0; n2 < t3.length; n2++)
          e3.push(t3[n2]);
        return e3.join("");
      }
      getErrorDisplayForChar(e3) {
        return e3.charCodeAt(0) === t2.EOF ? "<EOF>" : e3 === "\n" ? "\\n" : e3 === "	" ? "\\t" : e3 === "\r" ? "\\r" : e3;
      }
      getCharErrorDisplay(t3) {
        return "'" + this.getErrorDisplayForChar(t3) + "'";
      }
      recover(e3) {
        this._input.LA(1) !== t2.EOF && (e3 instanceof Mt ? this._interp.consume(this._input) : this._input.consume());
      }
      get inputStream() {
        return this._input;
      }
      set inputStream(t3) {
        this._input = null, this._tokenFactorySourcePair = [this, this._input], this.reset(), this._input = t3, this._tokenFactorySourcePair = [this, this._input];
      }
      get sourceName() {
        return this._input.sourceName;
      }
      get type() {
        return this._type;
      }
      set type(t3) {
        this._type = t3;
      }
      get line() {
        return this._interp.line;
      }
      set line(t3) {
        this._interp.line = t3;
      }
      get column() {
        return this._interp.column;
      }
      set column(t3) {
        this._interp.column = t3;
      }
      get text() {
        return this._text !== null ? this._text : this._interp.getText(this._input);
      }
      set text(t3) {
        this._text = t3;
      }
    }
    function Bt(t3) {
      return t3.hashCodeForConfigSet();
    }
    function Vt(t3, e3) {
      return t3 === e3 || t3 !== null && e3 !== null && t3.equalsForConfigSet(e3);
    }
    Ut.DEFAULT_MODE = 0, Ut.MORE = -2, Ut.SKIP = -3, Ut.DEFAULT_TOKEN_CHANNEL = t2.DEFAULT_CHANNEL, Ut.HIDDEN = t2.HIDDEN_CHANNEL, Ut.MIN_CHAR_VALUE = 0, Ut.MAX_CHAR_VALUE = 1114111;
    class zt {
      constructor(t3) {
        this.configLookup = new d2(Bt, Vt), this.fullCtx = t3 === void 0 || t3, this.readOnly = false, this.configs = [], this.uniqueAlt = 0, this.conflictingAlts = null, this.hasSemanticContext = false, this.dipsIntoOuterContext = false, this.cachedHashCode = -1;
      }
      add(t3, e3) {
        if (e3 === void 0 && (e3 = null), this.readOnly)
          throw "This set is readonly";
        t3.semanticContext !== p2.NONE && (this.hasSemanticContext = true), t3.reachesIntoOuterContext > 0 && (this.dipsIntoOuterContext = true);
        const n2 = this.configLookup.add(t3);
        if (n2 === t3)
          return this.cachedHashCode = -1, this.configs.push(t3), true;
        const s2 = !this.fullCtx, i3 = G(n2.context, t3.context, s2, e3);
        return n2.reachesIntoOuterContext = Math.max(n2.reachesIntoOuterContext, t3.reachesIntoOuterContext), t3.precedenceFilterSuppressed && (n2.precedenceFilterSuppressed = true), n2.context = i3, true;
      }
      getStates() {
        const t3 = new d2();
        for (let e3 = 0; e3 < this.configs.length; e3++)
          t3.add(this.configs[e3].state);
        return t3;
      }
      getPredicates() {
        const t3 = [];
        for (let e3 = 0; e3 < this.configs.length; e3++) {
          const n2 = this.configs[e3].semanticContext;
          n2 !== p2.NONE && t3.push(n2.semanticContext);
        }
        return t3;
      }
      optimizeConfigs(t3) {
        if (this.readOnly)
          throw "This set is readonly";
        if (this.configLookup.length !== 0)
          for (let e3 = 0; e3 < this.configs.length; e3++) {
            const n2 = this.configs[e3];
            n2.context = t3.getCachedContext(n2.context);
          }
      }
      addAll(t3) {
        for (let e3 = 0; e3 < t3.length; e3++)
          this.add(t3[e3]);
        return false;
      }
      equals(t3) {
        return this === t3 || t3 instanceof zt && e2(this.configs, t3.configs) && this.fullCtx === t3.fullCtx && this.uniqueAlt === t3.uniqueAlt && this.conflictingAlts === t3.conflictingAlts && this.hasSemanticContext === t3.hasSemanticContext && this.dipsIntoOuterContext === t3.dipsIntoOuterContext;
      }
      hashCode() {
        const t3 = new o2();
        return t3.update(this.configs), t3.finish();
      }
      updateHashCode(t3) {
        this.readOnly ? (this.cachedHashCode === -1 && (this.cachedHashCode = this.hashCode()), t3.update(this.cachedHashCode)) : t3.update(this.hashCode());
      }
      isEmpty() {
        return this.configs.length === 0;
      }
      contains(t3) {
        if (this.configLookup === null)
          throw "This method is not implemented for readonly sets.";
        return this.configLookup.contains(t3);
      }
      containsFast(t3) {
        if (this.configLookup === null)
          throw "This method is not implemented for readonly sets.";
        return this.configLookup.containsFast(t3);
      }
      clear() {
        if (this.readOnly)
          throw "This set is readonly";
        this.configs = [], this.cachedHashCode = -1, this.configLookup = new d2();
      }
      setReadonly(t3) {
        this.readOnly = t3, t3 && (this.configLookup = null);
      }
      toString() {
        return c2(this.configs) + (this.hasSemanticContext ? ",hasSemanticContext=" + this.hasSemanticContext : "") + (this.uniqueAlt !== $.INVALID_ALT_NUMBER ? ",uniqueAlt=" + this.uniqueAlt : "") + (this.conflictingAlts !== null ? ",conflictingAlts=" + this.conflictingAlts : "") + (this.dipsIntoOuterContext ? ",dipsIntoOuterContext" : "");
      }
      get items() {
        return this.configs;
      }
      get length() {
        return this.configs.length;
      }
    }
    class qt {
      constructor(t3, e3) {
        return t3 === null && (t3 = -1), e3 === null && (e3 = new zt()), this.stateNumber = t3, this.configs = e3, this.edges = null, this.isAcceptState = false, this.prediction = 0, this.lexerActionExecutor = null, this.requiresFullContext = false, this.predicates = null, this;
      }
      getAltSet() {
        const t3 = new d2();
        if (this.configs !== null)
          for (let e3 = 0; e3 < this.configs.length; e3++) {
            const n2 = this.configs[e3];
            t3.add(n2.alt);
          }
        return t3.length === 0 ? null : t3;
      }
      equals(t3) {
        return this === t3 || t3 instanceof qt && this.configs.equals(t3.configs);
      }
      toString() {
        let t3 = this.stateNumber + ":" + this.configs;
        return this.isAcceptState && (t3 += "=>", this.predicates !== null ? t3 += this.predicates : t3 += this.prediction), t3;
      }
      hashCode() {
        const t3 = new o2();
        return t3.update(this.configs), t3.finish();
      }
    }
    class Ht {
      constructor(t3, e3) {
        return this.atn = t3, this.sharedContextCache = e3, this;
      }
      getCachedContext(t3) {
        if (this.sharedContextCache === null)
          return t3;
        const e3 = new H();
        return Y(t3, this.sharedContextCache, e3);
      }
    }
    Ht.ERROR = new qt(2147483647, new zt());
    class Kt extends zt {
      constructor() {
        super(), this.configLookup = new d2();
      }
    }
    class Yt extends S2 {
      constructor(t3, e3) {
        super(t3, e3);
        const n2 = t3.lexerActionExecutor || null;
        return this.lexerActionExecutor = n2 || (e3 !== null ? e3.lexerActionExecutor : null), this.passedThroughNonGreedyDecision = e3 !== null && this.checkNonGreedyDecision(e3, this.state), this.hashCodeForConfigSet = Yt.prototype.hashCode, this.equalsForConfigSet = Yt.prototype.equals, this;
      }
      updateHashCode(t3) {
        t3.update(this.state.stateNumber, this.alt, this.context, this.semanticContext, this.passedThroughNonGreedyDecision, this.lexerActionExecutor);
      }
      equals(t3) {
        return this === t3 || t3 instanceof Yt && this.passedThroughNonGreedyDecision === t3.passedThroughNonGreedyDecision && (this.lexerActionExecutor ? this.lexerActionExecutor.equals(t3.lexerActionExecutor) : !t3.lexerActionExecutor) && super.equals(t3);
      }
      checkNonGreedyDecision(t3, e3) {
        return t3.passedThroughNonGreedyDecision || e3 instanceof J && e3.nonGreedy;
      }
    }
    class Gt extends St {
      constructor(t3, e3) {
        super(e3.actionType), this.offset = t3, this.action = e3, this.isPositionDependent = true;
      }
      execute(t3) {
        this.action.execute(t3);
      }
      updateHashCode(t3) {
        t3.update(this.actionType, this.offset, this.action);
      }
      equals(t3) {
        return this === t3 || t3 instanceof Gt && this.offset === t3.offset && this.action === t3.action;
      }
    }
    class jt {
      constructor(t3) {
        return this.lexerActions = t3 === null ? [] : t3, this.cachedHashCode = o2.hashStuff(t3), this;
      }
      fixOffsetBeforeMatch(t3) {
        let e3 = null;
        for (let n2 = 0; n2 < this.lexerActions.length; n2++)
          !this.lexerActions[n2].isPositionDependent || this.lexerActions[n2] instanceof Gt || (e3 === null && (e3 = this.lexerActions.concat([])), e3[n2] = new Gt(t3, this.lexerActions[n2]));
        return e3 === null ? this : new jt(e3);
      }
      execute(t3, e3, n2) {
        let s2 = false;
        const i3 = e3.index;
        try {
          for (let r3 = 0; r3 < this.lexerActions.length; r3++) {
            let o3 = this.lexerActions[r3];
            if (o3 instanceof Gt) {
              const t4 = o3.offset;
              e3.seek(n2 + t4), o3 = o3.action, s2 = n2 + t4 !== i3;
            } else
              o3.isPositionDependent && (e3.seek(i3), s2 = false);
            o3.execute(t3);
          }
        } finally {
          s2 && e3.seek(i3);
        }
      }
      hashCode() {
        return this.cachedHashCode;
      }
      updateHashCode(t3) {
        t3.update(this.cachedHashCode);
      }
      equals(t3) {
        if (this === t3)
          return true;
        if (t3 instanceof jt) {
          if (this.cachedHashCode != t3.cachedHashCode)
            return false;
          if (this.lexerActions.length != t3.lexerActions.length)
            return false;
          {
            const e3 = this.lexerActions.length;
            for (let n2 = 0; n2 < e3; ++n2)
              if (!this.lexerActions[n2].equals(t3.lexerActions[n2]))
                return false;
            return true;
          }
        }
        return false;
      }
      static append(t3, e3) {
        if (t3 === null)
          return new jt([e3]);
        const n2 = t3.lexerActions.concat([e3]);
        return new jt(n2);
      }
    }
    function Wt(t3) {
      t3.index = -1, t3.line = 0, t3.column = -1, t3.dfaState = null;
    }
    class $t {
      constructor() {
        Wt(this);
      }
      reset() {
        Wt(this);
      }
    }
    class Xt extends Ht {
      constructor(t3, e3, n2, s2) {
        super(e3, s2), this.decisionToDFA = n2, this.recog = t3, this.startIndex = -1, this.line = 1, this.column = 0, this.mode = Ut.DEFAULT_MODE, this.prevAccept = new $t();
      }
      copyState(t3) {
        this.column = t3.column, this.line = t3.line, this.mode = t3.mode, this.startIndex = t3.startIndex;
      }
      match(t3, e3) {
        this.mode = e3;
        const n2 = t3.mark();
        try {
          this.startIndex = t3.index, this.prevAccept.reset();
          const n3 = this.decisionToDFA[e3];
          return n3.s0 === null ? this.matchATN(t3) : this.execATN(t3, n3.s0);
        } finally {
          t3.release(n2);
        }
      }
      reset() {
        this.prevAccept.reset(), this.startIndex = -1, this.line = 1, this.column = 0, this.mode = Ut.DEFAULT_MODE;
      }
      matchATN(t3) {
        const e3 = this.atn.modeToStartState[this.mode];
        Xt.debug && console.log("matchATN mode " + this.mode + " start: " + e3);
        const n2 = this.mode, s2 = this.computeStartState(t3, e3), i3 = s2.hasSemanticContext;
        s2.hasSemanticContext = false;
        const r3 = this.addDFAState(s2);
        i3 || (this.decisionToDFA[this.mode].s0 = r3);
        const o3 = this.execATN(t3, r3);
        return Xt.debug && console.log("DFA after matchATN: " + this.decisionToDFA[n2].toLexerString()), o3;
      }
      execATN(e3, n2) {
        Xt.debug && console.log("start state closure=" + n2.configs), n2.isAcceptState && this.captureSimState(this.prevAccept, e3, n2);
        let s2 = e3.LA(1), i3 = n2;
        for (; ; ) {
          Xt.debug && console.log("execATN loop starting closure: " + i3.configs);
          let n3 = this.getExistingTargetState(i3, s2);
          if (n3 === null && (n3 = this.computeTargetState(e3, i3, s2)), n3 === Ht.ERROR)
            break;
          if (s2 !== t2.EOF && this.consume(e3), n3.isAcceptState && (this.captureSimState(this.prevAccept, e3, n3), s2 === t2.EOF))
            break;
          s2 = e3.LA(1), i3 = n3;
        }
        return this.failOrAccept(this.prevAccept, e3, i3.configs, s2);
      }
      getExistingTargetState(t3, e3) {
        if (t3.edges === null || e3 < Xt.MIN_DFA_EDGE || e3 > Xt.MAX_DFA_EDGE)
          return null;
        let n2 = t3.edges[e3 - Xt.MIN_DFA_EDGE];
        return n2 === void 0 && (n2 = null), Xt.debug && n2 !== null && console.log("reuse state " + t3.stateNumber + " edge to " + n2.stateNumber), n2;
      }
      computeTargetState(t3, e3, n2) {
        const s2 = new Kt();
        return this.getReachableConfigSet(t3, e3.configs, s2, n2), s2.items.length === 0 ? (s2.hasSemanticContext || this.addDFAEdge(e3, n2, Ht.ERROR), Ht.ERROR) : this.addDFAEdge(e3, n2, null, s2);
      }
      failOrAccept(e3, n2, s2, i3) {
        if (this.prevAccept.dfaState !== null) {
          const t3 = e3.dfaState.lexerActionExecutor;
          return this.accept(n2, t3, this.startIndex, e3.index, e3.line, e3.column), e3.dfaState.prediction;
        }
        if (i3 === t2.EOF && n2.index === this.startIndex)
          return t2.EOF;
        throw new Mt(this.recog, n2, this.startIndex, s2);
      }
      getReachableConfigSet(e3, n2, s2, i3) {
        let r3 = $.INVALID_ALT_NUMBER;
        for (let o3 = 0; o3 < n2.items.length; o3++) {
          const a3 = n2.items[o3], l3 = a3.alt === r3;
          if (!l3 || !a3.passedThroughNonGreedyDecision) {
            Xt.debug && console.log("testing %s at %s\n", this.getTokenName(i3), a3.toString(this.recog, true));
            for (let n3 = 0; n3 < a3.state.transitions.length; n3++) {
              const o4 = a3.state.transitions[n3], h3 = this.getReachableTarget(o4, i3);
              if (h3 !== null) {
                let n4 = a3.lexerActionExecutor;
                n4 !== null && (n4 = n4.fixOffsetBeforeMatch(e3.index - this.startIndex));
                const o5 = i3 === t2.EOF, c3 = new Yt({state: h3, lexerActionExecutor: n4}, a3);
                this.closure(e3, c3, s2, l3, true, o5) && (r3 = a3.alt);
              }
            }
          }
        }
      }
      accept(t3, e3, n2, s2, i3, r3) {
        Xt.debug && console.log("ACTION %s\n", e3), t3.seek(s2), this.line = i3, this.column = r3, e3 !== null && this.recog !== null && e3.execute(this.recog, t3, n2);
      }
      getReachableTarget(t3, e3) {
        return t3.matches(e3, 0, Ut.MAX_CHAR_VALUE) ? t3.target : null;
      }
      computeStartState(t3, e3) {
        const n2 = U.EMPTY, s2 = new Kt();
        for (let i3 = 0; i3 < e3.transitions.length; i3++) {
          const r3 = e3.transitions[i3].target, o3 = new Yt({state: r3, alt: i3 + 1, context: n2}, null);
          this.closure(t3, o3, s2, false, false, false);
        }
        return s2;
      }
      closure(t3, e3, n2, s2, i3, r3) {
        let o3 = null;
        if (Xt.debug && console.log("closure(" + e3.toString(this.recog, true) + ")"), e3.state instanceof A2) {
          if (Xt.debug && (this.recog !== null ? console.log("closure at %s rule stop %s\n", this.recog.ruleNames[e3.state.ruleIndex], e3) : console.log("closure at rule stop %s\n", e3)), e3.context === null || e3.context.hasEmptyPath()) {
            if (e3.context === null || e3.context.isEmpty())
              return n2.add(e3), true;
            n2.add(new Yt({state: e3.state, context: U.EMPTY}, e3)), s2 = true;
          }
          if (e3.context !== null && !e3.context.isEmpty()) {
            for (let a3 = 0; a3 < e3.context.length; a3++)
              if (e3.context.getReturnState(a3) !== U.EMPTY_RETURN_STATE) {
                const l3 = e3.context.getParent(a3), h3 = this.atn.states[e3.context.getReturnState(a3)];
                o3 = new Yt({state: h3, context: l3}, e3), s2 = this.closure(t3, o3, n2, s2, i3, r3);
              }
          }
          return s2;
        }
        e3.state.epsilonOnlyTransitions || s2 && e3.passedThroughNonGreedyDecision || n2.add(e3);
        for (let a3 = 0; a3 < e3.state.transitions.length; a3++) {
          const l3 = e3.state.transitions[a3];
          o3 = this.getEpsilonTarget(t3, e3, l3, n2, i3, r3), o3 !== null && (s2 = this.closure(t3, o3, n2, s2, i3, r3));
        }
        return s2;
      }
      getEpsilonTarget(e3, n2, s2, i3, r3, o3) {
        let a3 = null;
        if (s2.serializationType === C2.RULE) {
          const t3 = V.create(n2.context, s2.followState.stateNumber);
          a3 = new Yt({state: s2.target, context: t3}, n2);
        } else {
          if (s2.serializationType === C2.PRECEDENCE)
            throw "Precedence predicates are not supported in lexers.";
          if (s2.serializationType === C2.PREDICATE)
            Xt.debug && console.log("EVAL rule " + s2.ruleIndex + ":" + s2.predIndex), i3.hasSemanticContext = true, this.evaluatePredicate(e3, s2.ruleIndex, s2.predIndex, r3) && (a3 = new Yt({state: s2.target}, n2));
          else if (s2.serializationType === C2.ACTION)
            if (n2.context === null || n2.context.hasEmptyPath()) {
              const t3 = jt.append(n2.lexerActionExecutor, this.atn.lexerActions[s2.actionIndex]);
              a3 = new Yt({state: s2.target, lexerActionExecutor: t3}, n2);
            } else
              a3 = new Yt({state: s2.target}, n2);
          else
            s2.serializationType === C2.EPSILON ? a3 = new Yt({state: s2.target}, n2) : s2.serializationType !== C2.ATOM && s2.serializationType !== C2.RANGE && s2.serializationType !== C2.SET || o3 && s2.matches(t2.EOF, 0, Ut.MAX_CHAR_VALUE) && (a3 = new Yt({state: s2.target}, n2));
        }
        return a3;
      }
      evaluatePredicate(t3, e3, n2, s2) {
        if (this.recog === null)
          return true;
        if (!s2)
          return this.recog.sempred(null, e3, n2);
        const i3 = this.column, r3 = this.line, o3 = t3.index, a3 = t3.mark();
        try {
          return this.consume(t3), this.recog.sempred(null, e3, n2);
        } finally {
          this.column = i3, this.line = r3, t3.seek(o3), t3.release(a3);
        }
      }
      captureSimState(t3, e3, n2) {
        t3.index = e3.index, t3.line = this.line, t3.column = this.column, t3.dfaState = n2;
      }
      addDFAEdge(t3, e3, n2, s2) {
        if (n2 === void 0 && (n2 = null), s2 === void 0 && (s2 = null), n2 === null && s2 !== null) {
          const t4 = s2.hasSemanticContext;
          if (s2.hasSemanticContext = false, n2 = this.addDFAState(s2), t4)
            return n2;
        }
        return e3 < Xt.MIN_DFA_EDGE || e3 > Xt.MAX_DFA_EDGE || (Xt.debug && console.log("EDGE " + t3 + " -> " + n2 + " upon " + e3), t3.edges === null && (t3.edges = []), t3.edges[e3 - Xt.MIN_DFA_EDGE] = n2), n2;
      }
      addDFAState(t3) {
        const e3 = new qt(null, t3);
        let n2 = null;
        for (let e4 = 0; e4 < t3.items.length; e4++) {
          const s3 = t3.items[e4];
          if (s3.state instanceof A2) {
            n2 = s3;
            break;
          }
        }
        n2 !== null && (e3.isAcceptState = true, e3.lexerActionExecutor = n2.lexerActionExecutor, e3.prediction = this.atn.ruleToTokenType[n2.state.ruleIndex]);
        const s2 = this.decisionToDFA[this.mode], i3 = s2.states.get(e3);
        if (i3 !== null)
          return i3;
        const r3 = e3;
        return r3.stateNumber = s2.states.length, t3.setReadonly(true), r3.configs = t3, s2.states.add(r3), r3;
      }
      getDFA(t3) {
        return this.decisionToDFA[t3];
      }
      getText(t3) {
        return t3.getText(this.startIndex, t3.index - 1);
      }
      consume(t3) {
        t3.LA(1) === "\n".charCodeAt(0) ? (this.line += 1, this.column = 0) : this.column += 1, t3.consume();
      }
      getTokenName(t3) {
        return t3 === -1 ? "EOF" : "'" + String.fromCharCode(t3) + "'";
      }
    }
    Xt.debug = false, Xt.dfa_debug = false, Xt.MIN_DFA_EDGE = 0, Xt.MAX_DFA_EDGE = 127;
    class Jt {
      constructor(t3, e3) {
        this.alt = e3, this.pred = t3;
      }
      toString() {
        return "(" + this.pred + ", " + this.alt + ")";
      }
    }
    class Zt {
      constructor() {
        this.data = {};
      }
      get(t3) {
        return this.data["k-" + t3] || null;
      }
      set(t3, e3) {
        this.data["k-" + t3] = e3;
      }
      values() {
        return Object.keys(this.data).filter((t3) => t3.startsWith("k-")).map((t3) => this.data[t3], this);
      }
    }
    const Qt = {SLL: 0, LL: 1, LL_EXACT_AMBIG_DETECTION: 2, hasSLLConflictTerminatingPrediction: function(t3, e3) {
      if (Qt.allConfigsInRuleStopStates(e3))
        return true;
      if (t3 === Qt.SLL && e3.hasSemanticContext) {
        const t4 = new zt();
        for (let n3 = 0; n3 < e3.items.length; n3++) {
          let s2 = e3.items[n3];
          s2 = new S2({semanticContext: p2.NONE}, s2), t4.add(s2);
        }
        e3 = t4;
      }
      const n2 = Qt.getConflictingAltSubsets(e3);
      return Qt.hasConflictingAltSet(n2) && !Qt.hasStateAssociatedWithOneAlt(e3);
    }, hasConfigInRuleStopState: function(t3) {
      for (let e3 = 0; e3 < t3.items.length; e3++)
        if (t3.items[e3].state instanceof A2)
          return true;
      return false;
    }, allConfigsInRuleStopStates: function(t3) {
      for (let e3 = 0; e3 < t3.items.length; e3++)
        if (!(t3.items[e3].state instanceof A2))
          return false;
      return true;
    }, resolvesToJustOneViableAlt: function(t3) {
      return Qt.getSingleViableAlt(t3);
    }, allSubsetsConflict: function(t3) {
      return !Qt.hasNonConflictingAltSet(t3);
    }, hasNonConflictingAltSet: function(t3) {
      for (let e3 = 0; e3 < t3.length; e3++)
        if (t3[e3].length === 1)
          return true;
      return false;
    }, hasConflictingAltSet: function(t3) {
      for (let e3 = 0; e3 < t3.length; e3++)
        if (t3[e3].length > 1)
          return true;
      return false;
    }, allSubsetsEqual: function(t3) {
      let e3 = null;
      for (let n2 = 0; n2 < t3.length; n2++) {
        const s2 = t3[n2];
        if (e3 === null)
          e3 = s2;
        else if (s2 !== e3)
          return false;
      }
      return true;
    }, getUniqueAlt: function(t3) {
      const e3 = Qt.getAlts(t3);
      return e3.length === 1 ? e3.minValue() : $.INVALID_ALT_NUMBER;
    }, getAlts: function(t3) {
      const e3 = new j();
      return t3.map(function(t4) {
        e3.or(t4);
      }), e3;
    }, getConflictingAltSubsets: function(t3) {
      const e3 = new H();
      return e3.hashFunction = function(t4) {
        o2.hashStuff(t4.state.stateNumber, t4.context);
      }, e3.equalsFunction = function(t4, e4) {
        return t4.state.stateNumber === e4.state.stateNumber && t4.context.equals(e4.context);
      }, t3.items.map(function(t4) {
        let n2 = e3.get(t4);
        n2 === null && (n2 = new j(), e3.set(t4, n2)), n2.add(t4.alt);
      }), e3.getValues();
    }, getStateToAltMap: function(t3) {
      const e3 = new Zt();
      return t3.items.map(function(t4) {
        let n2 = e3.get(t4.state);
        n2 === null && (n2 = new j(), e3.set(t4.state, n2)), n2.add(t4.alt);
      }), e3;
    }, hasStateAssociatedWithOneAlt: function(t3) {
      const e3 = Qt.getStateToAltMap(t3).values();
      for (let t4 = 0; t4 < e3.length; t4++)
        if (e3[t4].length === 1)
          return true;
      return false;
    }, getSingleViableAlt: function(t3) {
      let e3 = null;
      for (let n2 = 0; n2 < t3.length; n2++) {
        const s2 = t3[n2].minValue();
        if (e3 === null)
          e3 = s2;
        else if (e3 !== s2)
          return $.INVALID_ALT_NUMBER;
      }
      return e3;
    }}, te = Qt;
    class ee extends Ft {
      constructor(t3, e3, n2, s2, i3, r3) {
        r3 = r3 || t3._ctx, s2 = s2 || t3.getCurrentToken(), n2 = n2 || t3.getCurrentToken(), e3 = e3 || t3.getInputStream(), super({message: "", recognizer: t3, input: e3, ctx: r3}), this.deadEndConfigs = i3, this.startToken = n2, this.offendingToken = s2;
      }
    }
    class ne {
      constructor(t3) {
        this.defaultMapCtor = t3 || H, this.cacheMap = new this.defaultMapCtor();
      }
      get(t3, e3) {
        const n2 = this.cacheMap.get(t3) || null;
        return n2 === null ? null : n2.get(e3) || null;
      }
      set(t3, e3, n2) {
        let s2 = this.cacheMap.get(t3) || null;
        s2 === null && (s2 = new this.defaultMapCtor(), this.cacheMap.set(t3, s2)), s2.set(e3, n2);
      }
    }
    class se extends Ht {
      constructor(t3, e3, n2, s2) {
        super(e3, s2), this.parser = t3, this.decisionToDFA = n2, this.predictionMode = te.LL, this._input = null, this._startIndex = 0, this._outerContext = null, this._dfa = null, this.mergeCache = null, this.debug = false, this.debug_closure = false, this.debug_add = false, this.trace_atn_sim = false, this.dfa_debug = false, this.retry_debug = false;
      }
      reset() {
      }
      adaptivePredict(t3, e3, n2) {
        (this.debug || this.trace_atn_sim) && console.log("adaptivePredict decision " + e3 + " exec LA(1)==" + this.getLookaheadName(t3) + " line " + t3.LT(1).line + ":" + t3.LT(1).column), this._input = t3, this._startIndex = t3.index, this._outerContext = n2;
        const s2 = this.decisionToDFA[e3];
        this._dfa = s2;
        const i3 = t3.mark(), r3 = t3.index;
        try {
          let e4;
          if (e4 = s2.precedenceDfa ? s2.getPrecedenceStartState(this.parser.getPrecedence()) : s2.s0, e4 === null) {
            n2 === null && (n2 = M2.EMPTY), this.debug && console.log("predictATN decision " + s2.decision + " exec LA(1)==" + this.getLookaheadName(t3) + ", outerContext=" + n2.toString(this.parser.ruleNames));
            const i5 = false;
            let r4 = this.computeStartState(s2.atnStartState, M2.EMPTY, i5);
            s2.precedenceDfa ? (s2.s0.configs = r4, r4 = this.applyPrecedenceFilter(r4), e4 = this.addDFAState(s2, new qt(null, r4)), s2.setPrecedenceStartState(this.parser.getPrecedence(), e4)) : (e4 = this.addDFAState(s2, new qt(null, r4)), s2.s0 = e4);
          }
          const i4 = this.execATN(s2, e4, t3, r3, n2);
          return this.debug && console.log("DFA after predictATN: " + s2.toString(this.parser.literalNames, this.parser.symbolicNames)), i4;
        } finally {
          this._dfa = null, this.mergeCache = null, t3.seek(r3), t3.release(i3);
        }
      }
      execATN(e3, n2, s2, i3, r3) {
        let o3;
        (this.debug || this.trace_atn_sim) && console.log("execATN decision " + e3.decision + ", DFA state " + n2 + ", LA(1)==" + this.getLookaheadName(s2) + " line " + s2.LT(1).line + ":" + s2.LT(1).column);
        let a3 = n2;
        this.debug && console.log("s0 = " + n2);
        let l3 = s2.LA(1);
        for (; ; ) {
          let n3 = this.getExistingTargetState(a3, l3);
          if (n3 === null && (n3 = this.computeTargetState(e3, a3, l3)), n3 === Ht.ERROR) {
            const t3 = this.noViableAlt(s2, r3, a3.configs, i3);
            if (s2.seek(i3), o3 = this.getSynValidOrSemInvalidAltThatFinishedDecisionEntryRule(a3.configs, r3), o3 !== $.INVALID_ALT_NUMBER)
              return o3;
            throw t3;
          }
          if (n3.requiresFullContext && this.predictionMode !== te.SLL) {
            let t3 = null;
            if (n3.predicates !== null) {
              this.debug && console.log("DFA state has preds in DFA sim LL failover");
              const e4 = s2.index;
              if (e4 !== i3 && s2.seek(i3), t3 = this.evalSemanticContext(n3.predicates, r3, true), t3.length === 1)
                return this.debug && console.log("Full LL avoided"), t3.minValue();
              e4 !== i3 && s2.seek(e4);
            }
            this.dfa_debug && console.log("ctx sensitive state " + r3 + " in " + n3);
            const a4 = true, l4 = this.computeStartState(e3.atnStartState, r3, a4);
            return this.reportAttemptingFullContext(e3, t3, n3.configs, i3, s2.index), o3 = this.execATNWithFullContext(e3, n3, l4, s2, i3, r3), o3;
          }
          if (n3.isAcceptState) {
            if (n3.predicates === null)
              return n3.prediction;
            const t3 = s2.index;
            s2.seek(i3);
            const o4 = this.evalSemanticContext(n3.predicates, r3, true);
            if (o4.length === 0)
              throw this.noViableAlt(s2, r3, n3.configs, i3);
            return o4.length === 1 || this.reportAmbiguity(e3, n3, i3, t3, false, o4, n3.configs), o4.minValue();
          }
          a3 = n3, l3 !== t2.EOF && (s2.consume(), l3 = s2.LA(1));
        }
      }
      getExistingTargetState(t3, e3) {
        const n2 = t3.edges;
        return n2 === null ? null : n2[e3 + 1] || null;
      }
      computeTargetState(t3, e3, n2) {
        const s2 = this.computeReachSet(e3.configs, n2, false);
        if (s2 === null)
          return this.addDFAEdge(t3, e3, n2, Ht.ERROR), Ht.ERROR;
        let i3 = new qt(null, s2);
        const r3 = this.getUniqueAlt(s2);
        if (this.debug) {
          const t4 = te.getConflictingAltSubsets(s2);
          console.log("SLL altSubSets=" + c2(t4) + ", configs=" + s2 + ", predict=" + r3 + ", allSubsetsConflict=" + te.allSubsetsConflict(t4) + ", conflictingAlts=" + this.getConflictingAlts(s2));
        }
        return r3 !== $.INVALID_ALT_NUMBER ? (i3.isAcceptState = true, i3.configs.uniqueAlt = r3, i3.prediction = r3) : te.hasSLLConflictTerminatingPrediction(this.predictionMode, s2) && (i3.configs.conflictingAlts = this.getConflictingAlts(s2), i3.requiresFullContext = true, i3.isAcceptState = true, i3.prediction = i3.configs.conflictingAlts.minValue()), i3.isAcceptState && i3.configs.hasSemanticContext && (this.predicateDFAState(i3, this.atn.getDecisionState(t3.decision)), i3.predicates !== null && (i3.prediction = $.INVALID_ALT_NUMBER)), i3 = this.addDFAEdge(t3, e3, n2, i3), i3;
      }
      predicateDFAState(t3, e3) {
        const n2 = e3.transitions.length, s2 = this.getConflictingAltsOrUniqueAlt(t3.configs), i3 = this.getPredsForAmbigAlts(s2, t3.configs, n2);
        i3 !== null ? (t3.predicates = this.getPredicatePredictions(s2, i3), t3.prediction = $.INVALID_ALT_NUMBER) : t3.prediction = s2.minValue();
      }
      execATNWithFullContext(e3, n2, s2, i3, r3, o3) {
        (this.debug || this.trace_atn_sim) && console.log("execATNWithFullContext " + s2);
        let a3, l3 = false, h3 = s2;
        i3.seek(r3);
        let c3 = i3.LA(1), u3 = -1;
        for (; ; ) {
          if (a3 = this.computeReachSet(h3, c3, true), a3 === null) {
            const t3 = this.noViableAlt(i3, o3, h3, r3);
            i3.seek(r3);
            const e5 = this.getSynValidOrSemInvalidAltThatFinishedDecisionEntryRule(h3, o3);
            if (e5 !== $.INVALID_ALT_NUMBER)
              return e5;
            throw t3;
          }
          const e4 = te.getConflictingAltSubsets(a3);
          if (this.debug && console.log("LL altSubSets=" + e4 + ", predict=" + te.getUniqueAlt(e4) + ", resolvesToJustOneViableAlt=" + te.resolvesToJustOneViableAlt(e4)), a3.uniqueAlt = this.getUniqueAlt(a3), a3.uniqueAlt !== $.INVALID_ALT_NUMBER) {
            u3 = a3.uniqueAlt;
            break;
          }
          if (this.predictionMode !== te.LL_EXACT_AMBIG_DETECTION) {
            if (u3 = te.resolvesToJustOneViableAlt(e4), u3 !== $.INVALID_ALT_NUMBER)
              break;
          } else if (te.allSubsetsConflict(e4) && te.allSubsetsEqual(e4)) {
            l3 = true, u3 = te.getSingleViableAlt(e4);
            break;
          }
          h3 = a3, c3 !== t2.EOF && (i3.consume(), c3 = i3.LA(1));
        }
        return a3.uniqueAlt !== $.INVALID_ALT_NUMBER ? (this.reportContextSensitivity(e3, u3, a3, r3, i3.index), u3) : (this.reportAmbiguity(e3, n2, r3, i3.index, l3, null, a3), u3);
      }
      computeReachSet(e3, n2, s2) {
        this.debug && console.log("in computeReachSet, starting closure: " + e3), this.mergeCache === null && (this.mergeCache = new ne());
        const i3 = new zt(s2);
        let r3 = null;
        for (let o4 = 0; o4 < e3.items.length; o4++) {
          const a3 = e3.items[o4];
          if (this.debug && console.log("testing " + this.getTokenName(n2) + " at " + a3), a3.state instanceof A2)
            (s2 || n2 === t2.EOF) && (r3 === null && (r3 = []), r3.push(a3), this.debug_add && console.log("added " + a3 + " to skippedStopStates"));
          else
            for (let t3 = 0; t3 < a3.state.transitions.length; t3++) {
              const e4 = a3.state.transitions[t3], s3 = this.getReachableTarget(e4, n2);
              if (s3 !== null) {
                const t4 = new S2({state: s3}, a3);
                i3.add(t4, this.mergeCache), this.debug_add && console.log("added " + t4 + " to intermediate");
              }
            }
        }
        let o3 = null;
        if (r3 === null && n2 !== t2.EOF && (i3.items.length === 1 || this.getUniqueAlt(i3) !== $.INVALID_ALT_NUMBER) && (o3 = i3), o3 === null) {
          o3 = new zt(s2);
          const e4 = new d2(), r4 = n2 === t2.EOF;
          for (let t3 = 0; t3 < i3.items.length; t3++)
            this.closure(i3.items[t3], o3, e4, false, s2, r4);
        }
        if (n2 === t2.EOF && (o3 = this.removeAllConfigsNotInRuleStopState(o3, o3 === i3)), !(r3 === null || s2 && te.hasConfigInRuleStopState(o3)))
          for (let t3 = 0; t3 < r3.length; t3++)
            o3.add(r3[t3], this.mergeCache);
        return this.trace_atn_sim && console.log("computeReachSet " + e3 + " -> " + o3), o3.items.length === 0 ? null : o3;
      }
      removeAllConfigsNotInRuleStopState(e3, n2) {
        if (te.allConfigsInRuleStopStates(e3))
          return e3;
        const s2 = new zt(e3.fullCtx);
        for (let i3 = 0; i3 < e3.items.length; i3++) {
          const r3 = e3.items[i3];
          if (r3.state instanceof A2)
            s2.add(r3, this.mergeCache);
          else if (n2 && r3.state.epsilonOnlyTransitions && this.atn.nextTokens(r3.state).contains(t2.EPSILON)) {
            const t3 = this.atn.ruleToStopState[r3.state.ruleIndex];
            s2.add(new S2({state: t3}, r3), this.mergeCache);
          }
        }
        return s2;
      }
      computeStartState(t3, e3, n2) {
        const s2 = K(this.atn, e3), i3 = new zt(n2);
        this.trace_atn_sim && console.log("computeStartState from ATN state " + t3 + " initialContext=" + s2.toString(this.parser));
        for (let e4 = 0; e4 < t3.transitions.length; e4++) {
          const r3 = t3.transitions[e4].target, o3 = new S2({state: r3, alt: e4 + 1, context: s2}, null), a3 = new d2();
          this.closure(o3, i3, a3, true, n2, false);
        }
        return i3;
      }
      applyPrecedenceFilter(t3) {
        let e3;
        const n2 = [], s2 = new zt(t3.fullCtx);
        for (let i3 = 0; i3 < t3.items.length; i3++) {
          if (e3 = t3.items[i3], e3.alt !== 1)
            continue;
          const r3 = e3.semanticContext.evalPrecedence(this.parser, this._outerContext);
          r3 !== null && (n2[e3.state.stateNumber] = e3.context, r3 !== e3.semanticContext ? s2.add(new S2({semanticContext: r3}, e3), this.mergeCache) : s2.add(e3, this.mergeCache));
        }
        for (let i3 = 0; i3 < t3.items.length; i3++)
          if (e3 = t3.items[i3], e3.alt !== 1) {
            if (!e3.precedenceFilterSuppressed) {
              const t4 = n2[e3.state.stateNumber] || null;
              if (t4 !== null && t4.equals(e3.context))
                continue;
            }
            s2.add(e3, this.mergeCache);
          }
        return s2;
      }
      getReachableTarget(t3, e3) {
        return t3.matches(e3, 0, this.atn.maxTokenType) ? t3.target : null;
      }
      getPredsForAmbigAlts(t3, e3, n2) {
        let s2 = [];
        for (let n3 = 0; n3 < e3.items.length; n3++) {
          const i4 = e3.items[n3];
          t3.has(i4.alt) && (s2[i4.alt] = p2.orContext(s2[i4.alt] || null, i4.semanticContext));
        }
        let i3 = 0;
        for (let t4 = 1; t4 < n2 + 1; t4++) {
          const e4 = s2[t4] || null;
          e4 === null ? s2[t4] = p2.NONE : e4 !== p2.NONE && (i3 += 1);
        }
        return i3 === 0 && (s2 = null), this.debug && console.log("getPredsForAmbigAlts result " + c2(s2)), s2;
      }
      getPredicatePredictions(t3, e3) {
        const n2 = [];
        let s2 = false;
        for (let i3 = 1; i3 < e3.length; i3++) {
          const r3 = e3[i3];
          t3 !== null && t3.has(i3) && n2.push(new Jt(r3, i3)), r3 !== p2.NONE && (s2 = true);
        }
        return s2 ? n2 : null;
      }
      getSynValidOrSemInvalidAltThatFinishedDecisionEntryRule(t3, e3) {
        const n2 = this.splitAccordingToSemanticValidity(t3, e3), s2 = n2[0], i3 = n2[1];
        let r3 = this.getAltThatFinishedDecisionEntryRule(s2);
        return r3 !== $.INVALID_ALT_NUMBER || i3.items.length > 0 && (r3 = this.getAltThatFinishedDecisionEntryRule(i3), r3 !== $.INVALID_ALT_NUMBER) ? r3 : $.INVALID_ALT_NUMBER;
      }
      getAltThatFinishedDecisionEntryRule(t3) {
        const e3 = [];
        for (let n2 = 0; n2 < t3.items.length; n2++) {
          const s2 = t3.items[n2];
          (s2.reachesIntoOuterContext > 0 || s2.state instanceof A2 && s2.context.hasEmptyPath()) && e3.indexOf(s2.alt) < 0 && e3.push(s2.alt);
        }
        return e3.length === 0 ? $.INVALID_ALT_NUMBER : Math.min.apply(null, e3);
      }
      splitAccordingToSemanticValidity(t3, e3) {
        const n2 = new zt(t3.fullCtx), s2 = new zt(t3.fullCtx);
        for (let i3 = 0; i3 < t3.items.length; i3++) {
          const r3 = t3.items[i3];
          r3.semanticContext !== p2.NONE ? r3.semanticContext.evaluate(this.parser, e3) ? n2.add(r3) : s2.add(r3) : n2.add(r3);
        }
        return [n2, s2];
      }
      evalSemanticContext(t3, e3, n2) {
        const s2 = new j();
        for (let i3 = 0; i3 < t3.length; i3++) {
          const r3 = t3[i3];
          if (r3.pred === p2.NONE) {
            if (s2.add(r3.alt), !n2)
              break;
            continue;
          }
          const o3 = r3.pred.evaluate(this.parser, e3);
          if ((this.debug || this.dfa_debug) && console.log("eval pred " + r3 + "=" + o3), o3 && ((this.debug || this.dfa_debug) && console.log("PREDICT " + r3.alt), s2.add(r3.alt), !n2))
            break;
        }
        return s2;
      }
      closure(t3, e3, n2, s2, i3, r3) {
        this.closureCheckingStopState(t3, e3, n2, s2, i3, 0, r3);
      }
      closureCheckingStopState(t3, e3, n2, s2, i3, r3, o3) {
        if ((this.trace_atn_sim || this.debug_closure) && console.log("closure(" + t3.toString(this.parser, true) + ")"), t3.state instanceof A2) {
          if (!t3.context.isEmpty()) {
            for (let a3 = 0; a3 < t3.context.length; a3++) {
              if (t3.context.getReturnState(a3) === U.EMPTY_RETURN_STATE) {
                if (i3) {
                  e3.add(new S2({state: t3.state, context: U.EMPTY}, t3), this.mergeCache);
                  continue;
                }
                this.debug && console.log("FALLING off rule " + this.getRuleName(t3.state.ruleIndex)), this.closure_(t3, e3, n2, s2, i3, r3, o3);
                continue;
              }
              const l3 = this.atn.states[t3.context.getReturnState(a3)], h3 = t3.context.getParent(a3), c3 = {state: l3, alt: t3.alt, context: h3, semanticContext: t3.semanticContext}, u3 = new S2(c3, null);
              u3.reachesIntoOuterContext = t3.reachesIntoOuterContext, this.closureCheckingStopState(u3, e3, n2, s2, i3, r3 - 1, o3);
            }
            return;
          }
          if (i3)
            return void e3.add(t3, this.mergeCache);
          this.debug && console.log("FALLING off rule " + this.getRuleName(t3.state.ruleIndex));
        }
        this.closure_(t3, e3, n2, s2, i3, r3, o3);
      }
      closure_(t3, e3, n2, s2, i3, r3, o3) {
        const a3 = t3.state;
        a3.epsilonOnlyTransitions || e3.add(t3, this.mergeCache);
        for (let l3 = 0; l3 < a3.transitions.length; l3++) {
          if (l3 === 0 && this.canDropLoopEntryEdgeInLeftRecursiveRule(t3))
            continue;
          const h3 = a3.transitions[l3], c3 = s2 && !(h3 instanceof ut), u3 = this.getEpsilonTarget(t3, h3, c3, r3 === 0, i3, o3);
          if (u3 !== null) {
            let s3 = r3;
            if (t3.state instanceof A2) {
              if (this._dfa !== null && this._dfa.precedenceDfa && h3.outermostPrecedenceReturn === this._dfa.atnStartState.ruleIndex && (u3.precedenceFilterSuppressed = true), u3.reachesIntoOuterContext += 1, n2.add(u3) !== u3)
                continue;
              e3.dipsIntoOuterContext = true, s3 -= 1, this.debug && console.log("dips into outer ctx: " + u3);
            } else {
              if (!h3.isEpsilon && n2.add(u3) !== u3)
                continue;
              h3 instanceof N2 && s3 >= 0 && (s3 += 1);
            }
            this.closureCheckingStopState(u3, e3, n2, c3, i3, s3, o3);
          }
        }
      }
      canDropLoopEntryEdgeInLeftRecursiveRule(t3) {
        const e3 = t3.state;
        if (e3.stateType !== _2.STAR_LOOP_ENTRY)
          return false;
        if (e3.stateType !== _2.STAR_LOOP_ENTRY || !e3.isPrecedenceDecision || t3.context.isEmpty() || t3.context.hasEmptyPath())
          return false;
        const n2 = t3.context.length;
        for (let s3 = 0; s3 < n2; s3++)
          if (this.atn.states[t3.context.getReturnState(s3)].ruleIndex !== e3.ruleIndex)
            return false;
        const s2 = e3.transitions[0].target.endState.stateNumber, i3 = this.atn.states[s2];
        for (let s3 = 0; s3 < n2; s3++) {
          const n3 = t3.context.getReturnState(s3), r3 = this.atn.states[n3];
          if (r3.transitions.length !== 1 || !r3.transitions[0].isEpsilon)
            return false;
          const o3 = r3.transitions[0].target;
          if (!(r3.stateType === _2.BLOCK_END && o3 === e3 || r3 === i3 || o3 === i3 || o3.stateType === _2.BLOCK_END && o3.transitions.length === 1 && o3.transitions[0].isEpsilon && o3.transitions[0].target === e3))
            return false;
        }
        return true;
      }
      getRuleName(t3) {
        return this.parser !== null && t3 >= 0 ? this.parser.ruleNames[t3] : "<rule " + t3 + ">";
      }
      getEpsilonTarget(e3, n2, s2, i3, r3, o3) {
        switch (n2.serializationType) {
          case C2.RULE:
            return this.ruleTransition(e3, n2);
          case C2.PRECEDENCE:
            return this.precedenceTransition(e3, n2, s2, i3, r3);
          case C2.PREDICATE:
            return this.predTransition(e3, n2, s2, i3, r3);
          case C2.ACTION:
            return this.actionTransition(e3, n2);
          case C2.EPSILON:
            return new S2({state: n2.target}, e3);
          case C2.ATOM:
          case C2.RANGE:
          case C2.SET:
            return o3 && n2.matches(t2.EOF, 0, 1) ? new S2({state: n2.target}, e3) : null;
          default:
            return null;
        }
      }
      actionTransition(t3, e3) {
        if (this.debug) {
          const t4 = e3.actionIndex === -1 ? 65535 : e3.actionIndex;
          console.log("ACTION edge " + e3.ruleIndex + ":" + t4);
        }
        return new S2({state: e3.target}, t3);
      }
      precedenceTransition(t3, e3, n2, s2, i3) {
        this.debug && (console.log("PRED (collectPredicates=" + n2 + ") " + e3.precedence + ">=_p, ctx dependent=true"), this.parser !== null && console.log("context surrounding pred is " + c2(this.parser.getRuleInvocationStack())));
        let r3 = null;
        if (n2 && s2)
          if (i3) {
            const n3 = this._input.index;
            this._input.seek(this._startIndex);
            const s3 = e3.getPredicate().evaluate(this.parser, this._outerContext);
            this._input.seek(n3), s3 && (r3 = new S2({state: e3.target}, t3));
          } else {
            const n3 = p2.andContext(t3.semanticContext, e3.getPredicate());
            r3 = new S2({state: e3.target, semanticContext: n3}, t3);
          }
        else
          r3 = new S2({state: e3.target}, t3);
        return this.debug && console.log("config from pred transition=" + r3), r3;
      }
      predTransition(t3, e3, n2, s2, i3) {
        this.debug && (console.log("PRED (collectPredicates=" + n2 + ") " + e3.ruleIndex + ":" + e3.predIndex + ", ctx dependent=" + e3.isCtxDependent), this.parser !== null && console.log("context surrounding pred is " + c2(this.parser.getRuleInvocationStack())));
        let r3 = null;
        if (n2 && (e3.isCtxDependent && s2 || !e3.isCtxDependent))
          if (i3) {
            const n3 = this._input.index;
            this._input.seek(this._startIndex);
            const s3 = e3.getPredicate().evaluate(this.parser, this._outerContext);
            this._input.seek(n3), s3 && (r3 = new S2({state: e3.target}, t3));
          } else {
            const n3 = p2.andContext(t3.semanticContext, e3.getPredicate());
            r3 = new S2({state: e3.target, semanticContext: n3}, t3);
          }
        else
          r3 = new S2({state: e3.target}, t3);
        return this.debug && console.log("config from pred transition=" + r3), r3;
      }
      ruleTransition(t3, e3) {
        this.debug && console.log("CALL rule " + this.getRuleName(e3.target.ruleIndex) + ", ctx=" + t3.context);
        const n2 = e3.followState, s2 = V.create(t3.context, n2.stateNumber);
        return new S2({state: e3.target, context: s2}, t3);
      }
      getConflictingAlts(t3) {
        const e3 = te.getConflictingAltSubsets(t3);
        return te.getAlts(e3);
      }
      getConflictingAltsOrUniqueAlt(t3) {
        let e3 = null;
        return t3.uniqueAlt !== $.INVALID_ALT_NUMBER ? (e3 = new j(), e3.add(t3.uniqueAlt)) : e3 = t3.conflictingAlts, e3;
      }
      getTokenName(e3) {
        if (e3 === t2.EOF)
          return "EOF";
        if (this.parser !== null && this.parser.literalNames !== null) {
          if (!(e3 >= this.parser.literalNames.length && e3 >= this.parser.symbolicNames.length))
            return (this.parser.literalNames[e3] || this.parser.symbolicNames[e3]) + "<" + e3 + ">";
          console.log(e3 + " ttype out of range: " + this.parser.literalNames), console.log("" + this.parser.getInputStream().getTokens());
        }
        return "" + e3;
      }
      getLookaheadName(t3) {
        return this.getTokenName(t3.LA(1));
      }
      dumpDeadEndConfigs(t3) {
        console.log("dead end configs: ");
        const e3 = t3.getDeadEndConfigs();
        for (let t4 = 0; t4 < e3.length; t4++) {
          const n2 = e3[t4];
          let s2 = "no edges";
          if (n2.state.transitions.length > 0) {
            const t5 = n2.state.transitions[0];
            t5 instanceof ht ? s2 = "Atom " + this.getTokenName(t5.label) : t5 instanceof y2 && (s2 = (t5 instanceof I2 ? "~" : "") + "Set " + t5.set);
          }
          console.error(n2.toString(this.parser, true) + ":" + s2);
        }
      }
      noViableAlt(t3, e3, n2, s2) {
        return new ee(this.parser, t3, t3.get(s2), t3.LT(1), n2, e3);
      }
      getUniqueAlt(t3) {
        let e3 = $.INVALID_ALT_NUMBER;
        for (let n2 = 0; n2 < t3.items.length; n2++) {
          const s2 = t3.items[n2];
          if (e3 === $.INVALID_ALT_NUMBER)
            e3 = s2.alt;
          else if (s2.alt !== e3)
            return $.INVALID_ALT_NUMBER;
        }
        return e3;
      }
      addDFAEdge(t3, e3, n2, s2) {
        if (this.debug && console.log("EDGE " + e3 + " -> " + s2 + " upon " + this.getTokenName(n2)), s2 === null)
          return null;
        if (s2 = this.addDFAState(t3, s2), e3 === null || n2 < -1 || n2 > this.atn.maxTokenType)
          return s2;
        if (e3.edges === null && (e3.edges = []), e3.edges[n2 + 1] = s2, this.debug) {
          const e4 = this.parser === null ? null : this.parser.literalNames, n3 = this.parser === null ? null : this.parser.symbolicNames;
          console.log("DFA=\n" + t3.toString(e4, n3));
        }
        return s2;
      }
      addDFAState(t3, e3) {
        if (e3 === Ht.ERROR)
          return e3;
        const n2 = t3.states.get(e3);
        return n2 !== null ? (this.trace_atn_sim && console.log("addDFAState " + e3 + " exists"), n2) : (e3.stateNumber = t3.states.length, e3.configs.readOnly || (e3.configs.optimizeConfigs(this), e3.configs.setReadonly(true)), this.trace_atn_sim && console.log("addDFAState new " + e3), t3.states.add(e3), this.debug && console.log("adding new DFA state: " + e3), e3);
      }
      reportAttemptingFullContext(t3, e3, n2, s2, i3) {
        if (this.debug || this.retry_debug) {
          const e4 = new m2(s2, i3 + 1);
          console.log("reportAttemptingFullContext decision=" + t3.decision + ":" + n2 + ", input=" + this.parser.getTokenStream().getText(e4));
        }
        this.parser !== null && this.parser.getErrorListenerDispatch().reportAttemptingFullContext(this.parser, t3, s2, i3, e3, n2);
      }
      reportContextSensitivity(t3, e3, n2, s2, i3) {
        if (this.debug || this.retry_debug) {
          const e4 = new m2(s2, i3 + 1);
          console.log("reportContextSensitivity decision=" + t3.decision + ":" + n2 + ", input=" + this.parser.getTokenStream().getText(e4));
        }
        this.parser !== null && this.parser.getErrorListenerDispatch().reportContextSensitivity(this.parser, t3, s2, i3, e3, n2);
      }
      reportAmbiguity(t3, e3, n2, s2, i3, r3, o3) {
        if (this.debug || this.retry_debug) {
          const t4 = new m2(n2, s2 + 1);
          console.log("reportAmbiguity " + r3 + ":" + o3 + ", input=" + this.parser.getTokenStream().getText(t4));
        }
        this.parser !== null && this.parser.getErrorListenerDispatch().reportAmbiguity(this.parser, t3, n2, s2, i3, r3, o3);
      }
    }
    class ie {
      constructor() {
        this.cache = new H();
      }
      add(t3) {
        if (t3 === U.EMPTY)
          return U.EMPTY;
        const e3 = this.cache.get(t3) || null;
        return e3 !== null ? e3 : (this.cache.set(t3, t3), t3);
      }
      get(t3) {
        return this.cache.get(t3) || null;
      }
      get length() {
        return this.cache.length;
      }
    }
    const re = {ATN: $, ATNDeserializer: Lt, LexerATNSimulator: Xt, ParserATNSimulator: se, PredictionMode: te, PredictionContextCache: ie};
    class oe {
      constructor(t3, e3, n2) {
        this.dfa = t3, this.literalNames = e3 || [], this.symbolicNames = n2 || [];
      }
      toString() {
        if (this.dfa.s0 === null)
          return null;
        let t3 = "";
        const e3 = this.dfa.sortedStates();
        for (let n2 = 0; n2 < e3.length; n2++) {
          const s2 = e3[n2];
          if (s2.edges !== null) {
            const e4 = s2.edges.length;
            for (let n3 = 0; n3 < e4; n3++) {
              const e5 = s2.edges[n3] || null;
              e5 !== null && e5.stateNumber !== 2147483647 && (t3 = t3.concat(this.getStateString(s2)), t3 = t3.concat("-"), t3 = t3.concat(this.getEdgeLabel(n3)), t3 = t3.concat("->"), t3 = t3.concat(this.getStateString(e5)), t3 = t3.concat("\n"));
            }
          }
        }
        return t3.length === 0 ? null : t3;
      }
      getEdgeLabel(t3) {
        return t3 === 0 ? "EOF" : this.literalNames !== null || this.symbolicNames !== null ? this.literalNames[t3 - 1] || this.symbolicNames[t3 - 1] : String.fromCharCode(t3 - 1);
      }
      getStateString(t3) {
        const e3 = (t3.isAcceptState ? ":" : "") + "s" + t3.stateNumber + (t3.requiresFullContext ? "^" : "");
        return t3.isAcceptState ? t3.predicates !== null ? e3 + "=>" + c2(t3.predicates) : e3 + "=>" + t3.prediction.toString() : e3;
      }
    }
    class ae extends oe {
      constructor(t3) {
        super(t3, null);
      }
      getEdgeLabel(t3) {
        return "'" + String.fromCharCode(t3) + "'";
      }
    }
    class le {
      constructor(t3, e3) {
        if (e3 === void 0 && (e3 = 0), this.atnStartState = t3, this.decision = e3, this._states = new d2(), this.s0 = null, this.precedenceDfa = false, t3 instanceof rt && t3.isPrecedenceDecision) {
          this.precedenceDfa = true;
          const t4 = new qt(null, new zt());
          t4.edges = [], t4.isAcceptState = false, t4.requiresFullContext = false, this.s0 = t4;
        }
      }
      getPrecedenceStartState(t3) {
        if (!this.precedenceDfa)
          throw "Only precedence DFAs may contain a precedence start state.";
        return t3 < 0 || t3 >= this.s0.edges.length ? null : this.s0.edges[t3] || null;
      }
      setPrecedenceStartState(t3, e3) {
        if (!this.precedenceDfa)
          throw "Only precedence DFAs may contain a precedence start state.";
        t3 < 0 || (this.s0.edges[t3] = e3);
      }
      setPrecedenceDfa(t3) {
        if (this.precedenceDfa !== t3) {
          if (this._states = new d2(), t3) {
            const t4 = new qt(null, new zt());
            t4.edges = [], t4.isAcceptState = false, t4.requiresFullContext = false, this.s0 = t4;
          } else
            this.s0 = null;
          this.precedenceDfa = t3;
        }
      }
      sortedStates() {
        return this._states.values().sort(function(t3, e3) {
          return t3.stateNumber - e3.stateNumber;
        });
      }
      toString(t3, e3) {
        return t3 = t3 || null, e3 = e3 || null, this.s0 === null ? "" : new oe(this, t3, e3).toString();
      }
      toLexerString() {
        return this.s0 === null ? "" : new ae(this).toString();
      }
      get states() {
        return this._states;
      }
    }
    const he = {DFA: le, DFASerializer: oe, LexerDFASerializer: ae, PredPrediction: Jt}, ce = {PredictionContext: U}, ue = {Interval: m2, IntervalSet: E2};
    class de {
      visitTerminal(t3) {
      }
      visitErrorNode(t3) {
      }
      enterEveryRule(t3) {
      }
      exitEveryRule(t3) {
      }
    }
    class pe {
      visit(t3) {
        return Array.isArray(t3) ? t3.map(function(t4) {
          return t4.accept(this);
        }, this) : t3.accept(this);
      }
      visitChildren(t3) {
        return t3.children ? this.visit(t3.children) : null;
      }
      visitTerminal(t3) {
      }
      visitErrorNode(t3) {
      }
    }
    class ge {
      walk(t3, e3) {
        if (e3 instanceof b2 || e3.isErrorNode !== void 0 && e3.isErrorNode())
          t3.visitErrorNode(e3);
        else if (e3 instanceof P2)
          t3.visitTerminal(e3);
        else {
          this.enterRule(t3, e3);
          for (let n2 = 0; n2 < e3.getChildCount(); n2++) {
            const s2 = e3.getChild(n2);
            this.walk(t3, s2);
          }
          this.exitRule(t3, e3);
        }
      }
      enterRule(t3, e3) {
        const n2 = e3.ruleContext;
        t3.enterEveryRule(n2), n2.enterRule(t3);
      }
      exitRule(t3, e3) {
        const n2 = e3.ruleContext;
        n2.exitRule(t3), t3.exitEveryRule(n2);
      }
    }
    ge.DEFAULT = new ge();
    const fe = {Trees: F2, RuleNode: w2, ErrorNode: b2, TerminalNode: P2, ParseTreeListener: de, ParseTreeVisitor: pe, ParseTreeWalker: ge};
    class xe extends Ft {
      constructor(t3) {
        super({message: "", recognizer: t3, input: t3.getInputStream(), ctx: t3._ctx}), this.offendingToken = t3.getCurrentToken();
      }
    }
    class Te extends Ft {
      constructor(t3, e3, n2) {
        super({message: Se(e3, n2 || null), recognizer: t3, input: t3.getInputStream(), ctx: t3._ctx});
        const s2 = t3._interp.atn.states[t3.state].transitions[0];
        s2 instanceof gt ? (this.ruleIndex = s2.ruleIndex, this.predicateIndex = s2.predIndex) : (this.ruleIndex = 0, this.predicateIndex = 0), this.predicate = e3, this.offendingToken = t3.getCurrentToken();
      }
    }
    function Se(t3, e3) {
      return e3 !== null ? e3 : "failed predicate: {" + t3 + "}?";
    }
    class me extends Ot {
      constructor(t3) {
        super(), t3 = t3 || true, this.exactOnly = t3;
      }
      reportAmbiguity(t3, e3, n2, s2, i3, r3, o3) {
        if (this.exactOnly && !i3)
          return;
        const a3 = "reportAmbiguity d=" + this.getDecisionDescription(t3, e3) + ": ambigAlts=" + this.getConflictingAlts(r3, o3) + ", input='" + t3.getTokenStream().getText(new m2(n2, s2)) + "'";
        t3.notifyErrorListeners(a3);
      }
      reportAttemptingFullContext(t3, e3, n2, s2, i3, r3) {
        const o3 = "reportAttemptingFullContext d=" + this.getDecisionDescription(t3, e3) + ", input='" + t3.getTokenStream().getText(new m2(n2, s2)) + "'";
        t3.notifyErrorListeners(o3);
      }
      reportContextSensitivity(t3, e3, n2, s2, i3, r3) {
        const o3 = "reportContextSensitivity d=" + this.getDecisionDescription(t3, e3) + ", input='" + t3.getTokenStream().getText(new m2(n2, s2)) + "'";
        t3.notifyErrorListeners(o3);
      }
      getDecisionDescription(t3, e3) {
        const n2 = e3.decision, s2 = e3.atnStartState.ruleIndex, i3 = t3.ruleNames;
        if (s2 < 0 || s2 >= i3.length)
          return "" + n2;
        const r3 = i3[s2] || null;
        return r3 === null || r3.length === 0 ? "" + n2 : `${n2} (${r3})`;
      }
      getConflictingAlts(t3, e3) {
        if (t3 !== null)
          return t3;
        const n2 = new j();
        for (let t4 = 0; t4 < e3.items.length; t4++)
          n2.add(e3.items[t4].alt);
        return `{${n2.values().join(", ")}}`;
      }
    }
    class Ee extends Error {
      constructor() {
        super(), Error.captureStackTrace(this, Ee);
      }
    }
    class _e {
      reset(t3) {
      }
      recoverInline(t3) {
      }
      recover(t3, e3) {
      }
      sync(t3) {
      }
      inErrorRecoveryMode(t3) {
      }
      reportError(t3) {
      }
    }
    class Ae extends _e {
      constructor() {
        super(), this.errorRecoveryMode = false, this.lastErrorIndex = -1, this.lastErrorStates = null, this.nextTokensContext = null, this.nextTokenState = 0;
      }
      reset(t3) {
        this.endErrorCondition(t3);
      }
      beginErrorCondition(t3) {
        this.errorRecoveryMode = true;
      }
      inErrorRecoveryMode(t3) {
        return this.errorRecoveryMode;
      }
      endErrorCondition(t3) {
        this.errorRecoveryMode = false, this.lastErrorStates = null, this.lastErrorIndex = -1;
      }
      reportMatch(t3) {
        this.endErrorCondition(t3);
      }
      reportError(t3, e3) {
        this.inErrorRecoveryMode(t3) || (this.beginErrorCondition(t3), e3 instanceof ee ? this.reportNoViableAlternative(t3, e3) : e3 instanceof xe ? this.reportInputMismatch(t3, e3) : e3 instanceof Te ? this.reportFailedPredicate(t3, e3) : (console.log("unknown recognition error type: " + e3.constructor.name), console.log(e3.stack), t3.notifyErrorListeners(e3.getOffendingToken(), e3.getMessage(), e3)));
      }
      recover(t3, e3) {
        this.lastErrorIndex === t3.getInputStream().index && this.lastErrorStates !== null && this.lastErrorStates.indexOf(t3.state) >= 0 && t3.consume(), this.lastErrorIndex = t3._input.index, this.lastErrorStates === null && (this.lastErrorStates = []), this.lastErrorStates.push(t3.state);
        const n2 = this.getErrorRecoverySet(t3);
        this.consumeUntil(t3, n2);
      }
      sync(e3) {
        if (this.inErrorRecoveryMode(e3))
          return;
        const n2 = e3._interp.atn.states[e3.state], s2 = e3.getTokenStream().LA(1), i3 = e3.atn.nextTokens(n2);
        if (i3.contains(s2))
          return this.nextTokensContext = null, void (this.nextTokenState = _2.INVALID_STATE_NUMBER);
        if (i3.contains(t2.EPSILON))
          this.nextTokensContext === null && (this.nextTokensContext = e3._ctx, this.nextTokensState = e3._stateNumber);
        else
          switch (n2.stateType) {
            case _2.BLOCK_START:
            case _2.STAR_BLOCK_START:
            case _2.PLUS_BLOCK_START:
            case _2.STAR_LOOP_ENTRY:
              if (this.singleTokenDeletion(e3) !== null)
                return;
              throw new xe(e3);
            case _2.PLUS_LOOP_BACK:
            case _2.STAR_LOOP_BACK: {
              this.reportUnwantedToken(e3);
              const t3 = new E2();
              t3.addSet(e3.getExpectedTokens());
              const n3 = t3.addSet(this.getErrorRecoverySet(e3));
              this.consumeUntil(e3, n3);
            }
          }
      }
      reportNoViableAlternative(e3, n2) {
        const s2 = e3.getTokenStream();
        let i3;
        i3 = s2 !== null ? n2.startToken.type === t2.EOF ? "<EOF>" : s2.getText(new m2(n2.startToken.tokenIndex, n2.offendingToken.tokenIndex)) : "<unknown input>";
        const r3 = "no viable alternative at input " + this.escapeWSAndQuote(i3);
        e3.notifyErrorListeners(r3, n2.offendingToken, n2);
      }
      reportInputMismatch(t3, e3) {
        const n2 = "mismatched input " + this.getTokenErrorDisplay(e3.offendingToken) + " expecting " + e3.getExpectedTokens().toString(t3.literalNames, t3.symbolicNames);
        t3.notifyErrorListeners(n2, e3.offendingToken, e3);
      }
      reportFailedPredicate(t3, e3) {
        const n2 = "rule " + t3.ruleNames[t3._ctx.ruleIndex] + " " + e3.message;
        t3.notifyErrorListeners(n2, e3.offendingToken, e3);
      }
      reportUnwantedToken(t3) {
        if (this.inErrorRecoveryMode(t3))
          return;
        this.beginErrorCondition(t3);
        const e3 = t3.getCurrentToken(), n2 = "extraneous input " + this.getTokenErrorDisplay(e3) + " expecting " + this.getExpectedTokens(t3).toString(t3.literalNames, t3.symbolicNames);
        t3.notifyErrorListeners(n2, e3, null);
      }
      reportMissingToken(t3) {
        if (this.inErrorRecoveryMode(t3))
          return;
        this.beginErrorCondition(t3);
        const e3 = t3.getCurrentToken(), n2 = "missing " + this.getExpectedTokens(t3).toString(t3.literalNames, t3.symbolicNames) + " at " + this.getTokenErrorDisplay(e3);
        t3.notifyErrorListeners(n2, e3, null);
      }
      recoverInline(t3) {
        const e3 = this.singleTokenDeletion(t3);
        if (e3 !== null)
          return t3.consume(), e3;
        if (this.singleTokenInsertion(t3))
          return this.getMissingSymbol(t3);
        throw new xe(t3);
      }
      singleTokenInsertion(t3) {
        const e3 = t3.getTokenStream().LA(1), n2 = t3._interp.atn, s2 = n2.states[t3.state].transitions[0].target;
        return !!n2.nextTokens(s2, t3._ctx).contains(e3) && (this.reportMissingToken(t3), true);
      }
      singleTokenDeletion(t3) {
        const e3 = t3.getTokenStream().LA(2);
        if (this.getExpectedTokens(t3).contains(e3)) {
          this.reportUnwantedToken(t3), t3.consume();
          const e4 = t3.getCurrentToken();
          return this.reportMatch(t3), e4;
        }
        return null;
      }
      getMissingSymbol(e3) {
        const n2 = e3.getCurrentToken(), s2 = this.getExpectedTokens(e3).first();
        let i3;
        i3 = s2 === t2.EOF ? "<missing EOF>" : "<missing " + e3.literalNames[s2] + ">";
        let r3 = n2;
        const o3 = e3.getTokenStream().LT(-1);
        return r3.type === t2.EOF && o3 !== null && (r3 = o3), e3.getTokenFactory().create(r3.source, s2, i3, t2.DEFAULT_CHANNEL, -1, -1, r3.line, r3.column);
      }
      getExpectedTokens(t3) {
        return t3.getExpectedTokens();
      }
      getTokenErrorDisplay(e3) {
        if (e3 === null)
          return "<no token>";
        let n2 = e3.text;
        return n2 === null && (n2 = e3.type === t2.EOF ? "<EOF>" : "<" + e3.type + ">"), this.escapeWSAndQuote(n2);
      }
      escapeWSAndQuote(t3) {
        return "'" + (t3 = (t3 = (t3 = t3.replace(/\n/g, "\\n")).replace(/\r/g, "\\r")).replace(/\t/g, "\\t")) + "'";
      }
      getErrorRecoverySet(e3) {
        const n2 = e3._interp.atn;
        let s2 = e3._ctx;
        const i3 = new E2();
        for (; s2 !== null && s2.invokingState >= 0; ) {
          const t3 = n2.states[s2.invokingState].transitions[0], e4 = n2.nextTokens(t3.followState);
          i3.addSet(e4), s2 = s2.parentCtx;
        }
        return i3.removeOne(t2.EPSILON), i3;
      }
      consumeUntil(e3, n2) {
        let s2 = e3.getTokenStream().LA(1);
        for (; s2 !== t2.EOF && !n2.contains(s2); )
          e3.consume(), s2 = e3.getTokenStream().LA(1);
      }
    }
    class Ce extends Ae {
      constructor() {
        super();
      }
      recover(t3, e3) {
        let n2 = t3._ctx;
        for (; n2 !== null; )
          n2.exception = e3, n2 = n2.parentCtx;
        throw new Ee(e3);
      }
      recoverInline(t3) {
        this.recover(t3, new xe(t3));
      }
      sync(t3) {
      }
    }
    const Ne = {RecognitionException: Ft, NoViableAltException: ee, LexerNoViableAltException: Mt, InputMismatchException: xe, FailedPredicateException: Te, DiagnosticErrorListener: me, BailErrorStrategy: Ce, DefaultErrorStrategy: Ae, ErrorListener: Ot};
    class ye {
      constructor(t3, e3) {
        if (this.name = "<empty>", this.strdata = t3, this.decodeToUnicodeCodePoints = e3 || false, this._index = 0, this.data = [], this.decodeToUnicodeCodePoints)
          for (let t4 = 0; t4 < this.strdata.length; ) {
            const e4 = this.strdata.codePointAt(t4);
            this.data.push(e4), t4 += e4 <= 65535 ? 1 : 2;
          }
        else {
          this.data = new Array(this.strdata.length);
          for (let t4 = 0; t4 < this.strdata.length; t4++)
            this.data[t4] = this.strdata.charCodeAt(t4);
        }
        this._size = this.data.length;
      }
      reset() {
        this._index = 0;
      }
      consume() {
        if (this._index >= this._size)
          throw "cannot consume EOF";
        this._index += 1;
      }
      LA(e3) {
        if (e3 === 0)
          return 0;
        e3 < 0 && (e3 += 1);
        const n2 = this._index + e3 - 1;
        return n2 < 0 || n2 >= this._size ? t2.EOF : this.data[n2];
      }
      LT(t3) {
        return this.LA(t3);
      }
      mark() {
        return -1;
      }
      release(t3) {
      }
      seek(t3) {
        t3 <= this._index ? this._index = t3 : this._index = Math.min(t3, this._size);
      }
      getText(t3, e3) {
        if (e3 >= this._size && (e3 = this._size - 1), t3 >= this._size)
          return "";
        if (this.decodeToUnicodeCodePoints) {
          let n2 = "";
          for (let s2 = t3; s2 <= e3; s2++)
            n2 += String.fromCodePoint(this.data[s2]);
          return n2;
        }
        return this.strdata.slice(t3, e3 + 1);
      }
      toString() {
        return this.strdata;
      }
      get index() {
        return this._index;
      }
      get size() {
        return this._size;
      }
    }
    class Ie extends ye {
      constructor(t3, e3) {
        super(t3, e3);
      }
    }
    var ke = n(92);
    const Le = typeof process != "undefined" && process.versions != null && process.versions.node != null;
    class Oe extends Ie {
      static fromPath(t3, e3, n2) {
        if (!Le)
          throw new Error("FileStream is only available when running in Node!");
        ke.readFile(t3, e3, function(t4, e4) {
          let s2 = null;
          e4 !== null && (s2 = new ye(e4, true)), n2(t4, s2);
        });
      }
      constructor(t3, e3, n2) {
        if (!Le)
          throw new Error("FileStream is only available when running in Node!");
        super(ke.readFileSync(t3, e3 || "utf-8"), n2), this.fileName = t3;
      }
    }
    const ve = {fromString: function(t3) {
      return new ye(t3, true);
    }, fromBlob: function(t3, e3, n2, s2) {
      const i3 = new window.FileReader();
      i3.onload = function(t4) {
        const e4 = new ye(t4.target.result, true);
        n2(e4);
      }, i3.onerror = s2, i3.readAsText(t3, e3);
    }, fromBuffer: function(t3, e3) {
      return new ye(t3.toString(e3), true);
    }, fromPath: function(t3, e3, n2) {
      Oe.fromPath(t3, e3, n2);
    }, fromPathSync: function(t3, e3) {
      return new Oe(t3, e3);
    }}, Re = {arrayToString: c2, stringToCharArray: function(t3) {
      let e3 = new Uint16Array(t3.length);
      for (let n2 = 0; n2 < t3.length; n2++)
        e3[n2] = t3.charCodeAt(n2);
      return e3;
    }};
    class we {
    }
    class Pe extends we {
      constructor(t3) {
        super(), this.tokenSource = t3, this.tokens = [], this.index = -1, this.fetchedEOF = false;
      }
      mark() {
        return 0;
      }
      release(t3) {
      }
      reset() {
        this.seek(0);
      }
      seek(t3) {
        this.lazyInit(), this.index = this.adjustSeekIndex(t3);
      }
      get size() {
        return this.tokens.length;
      }
      get(t3) {
        return this.lazyInit(), this.tokens[t3];
      }
      consume() {
        let e3 = false;
        if (e3 = this.index >= 0 && (this.fetchedEOF ? this.index < this.tokens.length - 1 : this.index < this.tokens.length), !e3 && this.LA(1) === t2.EOF)
          throw "cannot consume EOF";
        this.sync(this.index + 1) && (this.index = this.adjustSeekIndex(this.index + 1));
      }
      sync(t3) {
        const e3 = t3 - this.tokens.length + 1;
        return !(e3 > 0) || this.fetch(e3) >= e3;
      }
      fetch(e3) {
        if (this.fetchedEOF)
          return 0;
        for (let n2 = 0; n2 < e3; n2++) {
          const e4 = this.tokenSource.nextToken();
          if (e4.tokenIndex = this.tokens.length, this.tokens.push(e4), e4.type === t2.EOF)
            return this.fetchedEOF = true, n2 + 1;
        }
        return e3;
      }
      getTokens(e3, n2, s2) {
        if (s2 === void 0 && (s2 = null), e3 < 0 || n2 < 0)
          return null;
        this.lazyInit();
        const i3 = [];
        n2 >= this.tokens.length && (n2 = this.tokens.length - 1);
        for (let r3 = e3; r3 < n2; r3++) {
          const e4 = this.tokens[r3];
          if (e4.type === t2.EOF)
            break;
          (s2 === null || s2.contains(e4.type)) && i3.push(e4);
        }
        return i3;
      }
      LA(t3) {
        return this.LT(t3).type;
      }
      LB(t3) {
        return this.index - t3 < 0 ? null : this.tokens[this.index - t3];
      }
      LT(t3) {
        if (this.lazyInit(), t3 === 0)
          return null;
        if (t3 < 0)
          return this.LB(-t3);
        const e3 = this.index + t3 - 1;
        return this.sync(e3), e3 >= this.tokens.length ? this.tokens[this.tokens.length - 1] : this.tokens[e3];
      }
      adjustSeekIndex(t3) {
        return t3;
      }
      lazyInit() {
        this.index === -1 && this.setup();
      }
      setup() {
        this.sync(0), this.index = this.adjustSeekIndex(0);
      }
      setTokenSource(t3) {
        this.tokenSource = t3, this.tokens = [], this.index = -1, this.fetchedEOF = false;
      }
      nextTokenOnChannel(e3, n2) {
        if (this.sync(e3), e3 >= this.tokens.length)
          return -1;
        let s2 = this.tokens[e3];
        for (; s2.channel !== this.channel; ) {
          if (s2.type === t2.EOF)
            return -1;
          e3 += 1, this.sync(e3), s2 = this.tokens[e3];
        }
        return e3;
      }
      previousTokenOnChannel(t3, e3) {
        for (; t3 >= 0 && this.tokens[t3].channel !== e3; )
          t3 -= 1;
        return t3;
      }
      getHiddenTokensToRight(t3, e3) {
        if (e3 === void 0 && (e3 = -1), this.lazyInit(), t3 < 0 || t3 >= this.tokens.length)
          throw t3 + " not in 0.." + this.tokens.length - 1;
        const n2 = this.nextTokenOnChannel(t3 + 1, Ut.DEFAULT_TOKEN_CHANNEL), s2 = t3 + 1, i3 = n2 === -1 ? this.tokens.length - 1 : n2;
        return this.filterForChannel(s2, i3, e3);
      }
      getHiddenTokensToLeft(t3, e3) {
        if (e3 === void 0 && (e3 = -1), this.lazyInit(), t3 < 0 || t3 >= this.tokens.length)
          throw t3 + " not in 0.." + this.tokens.length - 1;
        const n2 = this.previousTokenOnChannel(t3 - 1, Ut.DEFAULT_TOKEN_CHANNEL);
        if (n2 === t3 - 1)
          return null;
        const s2 = n2 + 1, i3 = t3 - 1;
        return this.filterForChannel(s2, i3, e3);
      }
      filterForChannel(t3, e3, n2) {
        const s2 = [];
        for (let i3 = t3; i3 < e3 + 1; i3++) {
          const t4 = this.tokens[i3];
          n2 === -1 ? t4.channel !== Ut.DEFAULT_TOKEN_CHANNEL && s2.push(t4) : t4.channel === n2 && s2.push(t4);
        }
        return s2.length === 0 ? null : s2;
      }
      getSourceName() {
        return this.tokenSource.getSourceName();
      }
      getText(e3) {
        this.lazyInit(), this.fill(), e3 || (e3 = new m2(0, this.tokens.length - 1));
        let n2 = e3.start;
        n2 instanceof t2 && (n2 = n2.tokenIndex);
        let s2 = e3.stop;
        if (s2 instanceof t2 && (s2 = s2.tokenIndex), n2 === null || s2 === null || n2 < 0 || s2 < 0)
          return "";
        s2 >= this.tokens.length && (s2 = this.tokens.length - 1);
        let i3 = "";
        for (let e4 = n2; e4 < s2 + 1; e4++) {
          const n3 = this.tokens[e4];
          if (n3.type === t2.EOF)
            break;
          i3 += n3.text;
        }
        return i3;
      }
      fill() {
        for (this.lazyInit(); this.fetch(1e3) === 1e3; )
          ;
      }
    }
    Object.defineProperty(Pe, "size", {get: function() {
      return this.tokens.length;
    }});
    class be extends Pe {
      constructor(e3, n2) {
        super(e3), this.channel = n2 === void 0 ? t2.DEFAULT_CHANNEL : n2;
      }
      adjustSeekIndex(t3) {
        return this.nextTokenOnChannel(t3, this.channel);
      }
      LB(t3) {
        if (t3 === 0 || this.index - t3 < 0)
          return null;
        let e3 = this.index, n2 = 1;
        for (; n2 <= t3; )
          e3 = this.previousTokenOnChannel(e3 - 1, this.channel), n2 += 1;
        return e3 < 0 ? null : this.tokens[e3];
      }
      LT(t3) {
        if (this.lazyInit(), t3 === 0)
          return null;
        if (t3 < 0)
          return this.LB(-t3);
        let e3 = this.index, n2 = 1;
        for (; n2 < t3; )
          this.sync(e3 + 1) && (e3 = this.nextTokenOnChannel(e3 + 1, this.channel)), n2 += 1;
        return this.tokens[e3];
      }
      getNumberOfOnChannelTokens() {
        let e3 = 0;
        this.fill();
        for (let n2 = 0; n2 < this.tokens.length; n2++) {
          const s2 = this.tokens[n2];
          if (s2.channel === this.channel && (e3 += 1), s2.type === t2.EOF)
            break;
        }
        return e3;
      }
    }
    class De extends de {
      constructor(t3) {
        super(), this.parser = t3;
      }
      enterEveryRule(t3) {
        console.log("enter   " + this.parser.ruleNames[t3.ruleIndex] + ", LT(1)=" + this.parser._input.LT(1).text);
      }
      visitTerminal(t3) {
        console.log("consume " + t3.symbol + " rule " + this.parser.ruleNames[this.parser._ctx.ruleIndex]);
      }
      exitEveryRule(t3) {
        console.log("exit    " + this.parser.ruleNames[t3.ruleIndex] + ", LT(1)=" + this.parser._input.LT(1).text);
      }
    }
    class Fe extends wt {
      constructor(t3) {
        super(), this._input = null, this._errHandler = new Ae(), this._precedenceStack = [], this._precedenceStack.push(0), this._ctx = null, this.buildParseTrees = true, this._tracer = null, this._parseListeners = null, this._syntaxErrors = 0, this.setInputStream(t3);
      }
      reset() {
        this._input !== null && this._input.seek(0), this._errHandler.reset(this), this._ctx = null, this._syntaxErrors = 0, this.setTrace(false), this._precedenceStack = [], this._precedenceStack.push(0), this._interp !== null && this._interp.reset();
      }
      match(t3) {
        let e3 = this.getCurrentToken();
        return e3.type === t3 ? (this._errHandler.reportMatch(this), this.consume()) : (e3 = this._errHandler.recoverInline(this), this.buildParseTrees && e3.tokenIndex === -1 && this._ctx.addErrorNode(e3)), e3;
      }
      matchWildcard() {
        let t3 = this.getCurrentToken();
        return t3.type > 0 ? (this._errHandler.reportMatch(this), this.consume()) : (t3 = this._errHandler.recoverInline(this), this.buildParseTrees && t3.tokenIndex === -1 && this._ctx.addErrorNode(t3)), t3;
      }
      getParseListeners() {
        return this._parseListeners || [];
      }
      addParseListener(t3) {
        if (t3 === null)
          throw "listener";
        this._parseListeners === null && (this._parseListeners = []), this._parseListeners.push(t3);
      }
      removeParseListener(t3) {
        if (this._parseListeners !== null) {
          const e3 = this._parseListeners.indexOf(t3);
          e3 >= 0 && this._parseListeners.splice(e3, 1), this._parseListeners.length === 0 && (this._parseListeners = null);
        }
      }
      removeParseListeners() {
        this._parseListeners = null;
      }
      triggerEnterRuleEvent() {
        if (this._parseListeners !== null) {
          const t3 = this._ctx;
          this._parseListeners.forEach(function(e3) {
            e3.enterEveryRule(t3), t3.enterRule(e3);
          });
        }
      }
      triggerExitRuleEvent() {
        if (this._parseListeners !== null) {
          const t3 = this._ctx;
          this._parseListeners.slice(0).reverse().forEach(function(e3) {
            t3.exitRule(e3), e3.exitEveryRule(t3);
          });
        }
      }
      getTokenFactory() {
        return this._input.tokenSource._factory;
      }
      setTokenFactory(t3) {
        this._input.tokenSource._factory = t3;
      }
      getATNWithBypassAlts() {
        const t3 = this.getSerializedATN();
        if (t3 === null)
          throw "The current parser does not support an ATN with bypass alternatives.";
        let e3 = this.bypassAltsAtnCache[t3];
        if (e3 === null) {
          const n2 = new Tt();
          n2.generateRuleBypassTransitions = true, e3 = new Lt(n2).deserialize(t3), this.bypassAltsAtnCache[t3] = e3;
        }
        return e3;
      }
      getInputStream() {
        return this.getTokenStream();
      }
      setInputStream(t3) {
        this.setTokenStream(t3);
      }
      getTokenStream() {
        return this._input;
      }
      setTokenStream(t3) {
        this._input = null, this.reset(), this._input = t3;
      }
      get syntaxErrorsCount() {
        return this._syntaxErrors;
      }
      getCurrentToken() {
        return this._input.LT(1);
      }
      notifyErrorListeners(t3, e3, n2) {
        n2 = n2 || null, (e3 = e3 || null) === null && (e3 = this.getCurrentToken()), this._syntaxErrors += 1;
        const s2 = e3.line, i3 = e3.column;
        this.getErrorListenerDispatch().syntaxError(this, e3, s2, i3, t3, n2);
      }
      consume() {
        const e3 = this.getCurrentToken();
        e3.type !== t2.EOF && this.getInputStream().consume();
        const n2 = this._parseListeners !== null && this._parseListeners.length > 0;
        if (this.buildParseTrees || n2) {
          let t3;
          t3 = this._errHandler.inErrorRecoveryMode(this) ? this._ctx.addErrorNode(e3) : this._ctx.addTokenNode(e3), t3.invokingState = this.state, n2 && this._parseListeners.forEach(function(e4) {
            t3 instanceof b2 || t3.isErrorNode !== void 0 && t3.isErrorNode() ? e4.visitErrorNode(t3) : t3 instanceof P2 && e4.visitTerminal(t3);
          });
        }
        return e3;
      }
      addContextToParseTree() {
        this._ctx.parentCtx !== null && this._ctx.parentCtx.addChild(this._ctx);
      }
      enterRule(t3, e3, n2) {
        this.state = e3, this._ctx = t3, this._ctx.start = this._input.LT(1), this.buildParseTrees && this.addContextToParseTree(), this.triggerEnterRuleEvent();
      }
      exitRule() {
        this._ctx.stop = this._input.LT(-1), this.triggerExitRuleEvent(), this.state = this._ctx.invokingState, this._ctx = this._ctx.parentCtx;
      }
      enterOuterAlt(t3, e3) {
        t3.setAltNumber(e3), this.buildParseTrees && this._ctx !== t3 && this._ctx.parentCtx !== null && (this._ctx.parentCtx.removeLastChild(), this._ctx.parentCtx.addChild(t3)), this._ctx = t3;
      }
      getPrecedence() {
        return this._precedenceStack.length === 0 ? -1 : this._precedenceStack[this._precedenceStack.length - 1];
      }
      enterRecursionRule(t3, e3, n2, s2) {
        this.state = e3, this._precedenceStack.push(s2), this._ctx = t3, this._ctx.start = this._input.LT(1), this.triggerEnterRuleEvent();
      }
      pushNewRecursionContext(t3, e3, n2) {
        const s2 = this._ctx;
        s2.parentCtx = t3, s2.invokingState = e3, s2.stop = this._input.LT(-1), this._ctx = t3, this._ctx.start = s2.start, this.buildParseTrees && this._ctx.addChild(s2), this.triggerEnterRuleEvent();
      }
      unrollRecursionContexts(t3) {
        this._precedenceStack.pop(), this._ctx.stop = this._input.LT(-1);
        const e3 = this._ctx, n2 = this.getParseListeners();
        if (n2 !== null && n2.length > 0)
          for (; this._ctx !== t3; )
            this.triggerExitRuleEvent(), this._ctx = this._ctx.parentCtx;
        else
          this._ctx = t3;
        e3.parentCtx = t3, this.buildParseTrees && t3 !== null && t3.addChild(e3);
      }
      getInvokingContext(t3) {
        let e3 = this._ctx;
        for (; e3 !== null; ) {
          if (e3.ruleIndex === t3)
            return e3;
          e3 = e3.parentCtx;
        }
        return null;
      }
      precpred(t3, e3) {
        return e3 >= this._precedenceStack[this._precedenceStack.length - 1];
      }
      inContext(t3) {
        return false;
      }
      isExpectedToken(e3) {
        const n2 = this._interp.atn;
        let s2 = this._ctx;
        const i3 = n2.states[this.state];
        let r3 = n2.nextTokens(i3);
        if (r3.contains(e3))
          return true;
        if (!r3.contains(t2.EPSILON))
          return false;
        for (; s2 !== null && s2.invokingState >= 0 && r3.contains(t2.EPSILON); ) {
          const t3 = n2.states[s2.invokingState].transitions[0];
          if (r3 = n2.nextTokens(t3.followState), r3.contains(e3))
            return true;
          s2 = s2.parentCtx;
        }
        return !(!r3.contains(t2.EPSILON) || e3 !== t2.EOF);
      }
      getExpectedTokens() {
        return this._interp.atn.getExpectedTokens(this.state, this._ctx);
      }
      getExpectedTokensWithinCurrentRule() {
        const t3 = this._interp.atn, e3 = t3.states[this.state];
        return t3.nextTokens(e3);
      }
      getRuleIndex(t3) {
        const e3 = this.getRuleIndexMap()[t3];
        return e3 !== null ? e3 : -1;
      }
      getRuleInvocationStack(t3) {
        (t3 = t3 || null) === null && (t3 = this._ctx);
        const e3 = [];
        for (; t3 !== null; ) {
          const n2 = t3.ruleIndex;
          n2 < 0 ? e3.push("n/a") : e3.push(this.ruleNames[n2]), t3 = t3.parentCtx;
        }
        return e3;
      }
      getDFAStrings() {
        return this._interp.decisionToDFA.toString();
      }
      dumpDFA() {
        let t3 = false;
        for (let e3 = 0; e3 < this._interp.decisionToDFA.length; e3++) {
          const n2 = this._interp.decisionToDFA[e3];
          n2.states.length > 0 && (t3 && console.log(), this.printer.println("Decision " + n2.decision + ":"), this.printer.print(n2.toString(this.literalNames, this.symbolicNames)), t3 = true);
        }
      }
      getSourceName() {
        return this._input.sourceName;
      }
      setTrace(t3) {
        t3 ? (this._tracer !== null && this.removeParseListener(this._tracer), this._tracer = new De(this), this.addParseListener(this._tracer)) : (this.removeParseListener(this._tracer), this._tracer = null);
      }
    }
    Fe.bypassAltsAtnCache = {};
    class Me extends P2 {
      constructor(t3) {
        super(), this.parentCtx = null, this.symbol = t3;
      }
      getChild(t3) {
        return null;
      }
      getSymbol() {
        return this.symbol;
      }
      getParent() {
        return this.parentCtx;
      }
      getPayload() {
        return this.symbol;
      }
      getSourceInterval() {
        if (this.symbol === null)
          return m2.INVALID_INTERVAL;
        const t3 = this.symbol.tokenIndex;
        return new m2(t3, t3);
      }
      getChildCount() {
        return 0;
      }
      accept(t3) {
        return t3.visitTerminal(this);
      }
      getText() {
        return this.symbol.text;
      }
      toString() {
        return this.symbol.type === t2.EOF ? "<EOF>" : this.symbol.text;
      }
    }
    class Ue extends Me {
      constructor(t3) {
        super(t3);
      }
      isErrorNode() {
        return true;
      }
      accept(t3) {
        return t3.visitErrorNode(this);
      }
    }
    class Be extends M2 {
      constructor(t3, e3) {
        super(t3, e3), this.children = null, this.start = null, this.stop = null, this.exception = null;
      }
      copyFrom(t3) {
        this.parentCtx = t3.parentCtx, this.invokingState = t3.invokingState, this.children = null, this.start = t3.start, this.stop = t3.stop, t3.children && (this.children = [], t3.children.map(function(t4) {
          t4 instanceof Ue && (this.children.push(t4), t4.parentCtx = this);
        }, this));
      }
      enterRule(t3) {
      }
      exitRule(t3) {
      }
      addChild(t3) {
        return this.children === null && (this.children = []), this.children.push(t3), t3;
      }
      removeLastChild() {
        this.children !== null && this.children.pop();
      }
      addTokenNode(t3) {
        const e3 = new Me(t3);
        return this.addChild(e3), e3.parentCtx = this, e3;
      }
      addErrorNode(t3) {
        const e3 = new Ue(t3);
        return this.addChild(e3), e3.parentCtx = this, e3;
      }
      getChild(t3, e3) {
        if (e3 = e3 || null, this.children === null || t3 < 0 || t3 >= this.children.length)
          return null;
        if (e3 === null)
          return this.children[t3];
        for (let n2 = 0; n2 < this.children.length; n2++) {
          const s2 = this.children[n2];
          if (s2 instanceof e3) {
            if (t3 === 0)
              return s2;
            t3 -= 1;
          }
        }
        return null;
      }
      getToken(t3, e3) {
        if (this.children === null || e3 < 0 || e3 >= this.children.length)
          return null;
        for (let n2 = 0; n2 < this.children.length; n2++) {
          const s2 = this.children[n2];
          if (s2 instanceof P2 && s2.symbol.type === t3) {
            if (e3 === 0)
              return s2;
            e3 -= 1;
          }
        }
        return null;
      }
      getTokens(t3) {
        if (this.children === null)
          return [];
        {
          const e3 = [];
          for (let n2 = 0; n2 < this.children.length; n2++) {
            const s2 = this.children[n2];
            s2 instanceof P2 && s2.symbol.type === t3 && e3.push(s2);
          }
          return e3;
        }
      }
      getTypedRuleContext(t3, e3) {
        return this.getChild(e3, t3);
      }
      getTypedRuleContexts(t3) {
        if (this.children === null)
          return [];
        {
          const e3 = [];
          for (let n2 = 0; n2 < this.children.length; n2++) {
            const s2 = this.children[n2];
            s2 instanceof t3 && e3.push(s2);
          }
          return e3;
        }
      }
      getChildCount() {
        return this.children === null ? 0 : this.children.length;
      }
      getSourceInterval() {
        return this.start === null || this.stop === null ? m2.INVALID_INTERVAL : new m2(this.start.tokenIndex, this.stop.tokenIndex);
      }
    }
    M2.EMPTY = new Be();
    class Ve {
      constructor(t3) {
        this.tokens = t3, this.programs = new Map();
      }
      getTokenStream() {
        return this.tokens;
      }
      insertAfter(t3, e3) {
        let n2, s2 = arguments.length > 2 && arguments[2] !== void 0 ? arguments[2] : Ve.DEFAULT_PROGRAM_NAME;
        n2 = typeof t3 == "number" ? t3 : t3.tokenIndex;
        let i3 = this.getProgram(s2), r3 = new Ge(this.tokens, n2, i3.length, e3);
        i3.push(r3);
      }
      insertBefore(t3, e3) {
        let n2, s2 = arguments.length > 2 && arguments[2] !== void 0 ? arguments[2] : Ve.DEFAULT_PROGRAM_NAME;
        n2 = typeof t3 == "number" ? t3 : t3.tokenIndex;
        const i3 = this.getProgram(s2), r3 = new Ye(this.tokens, n2, i3.length, e3);
        i3.push(r3);
      }
      replaceSingle(t3, e3) {
        let n2 = arguments.length > 2 && arguments[2] !== void 0 ? arguments[2] : Ve.DEFAULT_PROGRAM_NAME;
        this.replace(t3, t3, e3, n2);
      }
      replace(t3, e3, n2) {
        let s2 = arguments.length > 3 && arguments[3] !== void 0 ? arguments[3] : Ve.DEFAULT_PROGRAM_NAME;
        if (typeof t3 != "number" && (t3 = t3.tokenIndex), typeof e3 != "number" && (e3 = e3.tokenIndex), t3 > e3 || t3 < 0 || e3 < 0 || e3 >= this.tokens.size)
          throw new RangeError(`replace: range invalid: ${t3}..${e3}(size=${this.tokens.size})`);
        let i3 = this.getProgram(s2), r3 = new je(this.tokens, t3, e3, i3.length, n2);
        i3.push(r3);
      }
      delete(t3, e3) {
        let n2 = arguments.length > 2 && arguments[2] !== void 0 ? arguments[2] : Ve.DEFAULT_PROGRAM_NAME;
        e3 === void 0 && (e3 = t3), this.replace(t3, e3, null, n2);
      }
      getProgram(t3) {
        let e3 = this.programs.get(t3);
        return e3 == null && (e3 = this.initializeProgram(t3)), e3;
      }
      initializeProgram(t3) {
        const e3 = [];
        return this.programs.set(t3, e3), e3;
      }
      getText(e3) {
        let n2, s2 = arguments.length > 1 && arguments[1] !== void 0 ? arguments[1] : Ve.DEFAULT_PROGRAM_NAME;
        n2 = e3 instanceof m2 ? e3 : new m2(0, this.tokens.size - 1), typeof e3 == "string" && (s2 = e3);
        const i3 = this.programs.get(s2);
        let r3 = n2.start, o3 = n2.stop;
        if (o3 > this.tokens.size - 1 && (o3 = this.tokens.size - 1), r3 < 0 && (r3 = 0), i3 == null || i3.length === 0)
          return this.tokens.getText(new m2(r3, o3));
        let a3 = [], l3 = this.reduceToSingleOperationPerIndex(i3), h3 = r3;
        for (; h3 <= o3 && h3 < this.tokens.size; ) {
          let e4 = l3.get(h3);
          l3.delete(h3);
          let n3 = this.tokens.get(h3);
          e4 == null ? (n3.type !== t2.EOF && a3.push(String(n3.text)), h3++) : h3 = e4.execute(a3);
        }
        if (o3 === this.tokens.size - 1)
          for (const t3 of l3.values())
            t3.index >= this.tokens.size - 1 && a3.push(t3.text.toString());
        return a3.join("");
      }
      reduceToSingleOperationPerIndex(t3) {
        for (let e4 = 0; e4 < t3.length; e4++) {
          let n2 = t3[e4];
          if (n2 == null)
            continue;
          if (!(n2 instanceof je))
            continue;
          let s2 = n2, i3 = this.getKindOfOps(t3, Ye, e4);
          for (let e5 of i3)
            e5.index === s2.index ? (t3[e5.instructionIndex] = void 0, s2.text = e5.text.toString() + (s2.text != null ? s2.text.toString() : "")) : e5.index > s2.index && e5.index <= s2.lastIndex && (t3[e5.instructionIndex] = void 0);
          let r3 = this.getKindOfOps(t3, je, e4);
          for (let e5 of r3) {
            if (e5.index >= s2.index && e5.lastIndex <= s2.lastIndex) {
              t3[e5.instructionIndex] = void 0;
              continue;
            }
            let n3 = e5.lastIndex < s2.index || e5.index > s2.lastIndex;
            if (e5.text != null || s2.text != null || n3) {
              if (!n3)
                throw new Error(`replace op boundaries of ${s2} overlap with previous ${e5}`);
            } else
              t3[e5.instructionIndex] = void 0, s2.index = Math.min(e5.index, s2.index), s2.lastIndex = Math.max(e5.lastIndex, s2.lastIndex);
          }
        }
        for (let e4 = 0; e4 < t3.length; e4++) {
          let n2 = t3[e4];
          if (n2 == null)
            continue;
          if (!(n2 instanceof Ye))
            continue;
          let s2 = n2, i3 = this.getKindOfOps(t3, Ye, e4);
          for (let e5 of i3)
            e5.index === s2.index && (e5 instanceof Ge ? (s2.text = this.catOpText(e5.text, s2.text), t3[e5.instructionIndex] = void 0) : e5 instanceof Ye && (s2.text = this.catOpText(s2.text, e5.text), t3[e5.instructionIndex] = void 0));
          let r3 = this.getKindOfOps(t3, je, e4);
          for (let n3 of r3)
            if (s2.index !== n3.index) {
              if (s2.index >= n3.index && s2.index <= n3.lastIndex)
                throw new Error(`insert op ${s2} within boundaries of previous ${n3}`);
            } else
              n3.text = this.catOpText(s2.text, n3.text), t3[e4] = void 0;
        }
        let e3 = new Map();
        for (let n2 of t3)
          if (n2 != null) {
            if (e3.get(n2.index) != null)
              throw new Error("should only be one op per index");
            e3.set(n2.index, n2);
          }
        return e3;
      }
      catOpText(t3, e3) {
        let n2 = "", s2 = "";
        return t3 != null && (n2 = t3.toString()), e3 != null && (s2 = e3.toString()), n2 + s2;
      }
      getKindOfOps(t3, e3, n2) {
        return t3.slice(0, n2).filter((t4) => t4 && t4 instanceof e3);
      }
    }
    var ze, qe, He;
    ze = Ve, He = "default", (qe = function(t3) {
      var e3 = function(t4, e4) {
        if (typeof t4 != "object" || t4 === null)
          return t4;
        var n2 = t4[Symbol.toPrimitive];
        if (n2 !== void 0) {
          var s2 = n2.call(t4, "string");
          if (typeof s2 != "object")
            return s2;
          throw new TypeError("@@toPrimitive must return a primitive value.");
        }
        return String(t4);
      }(t3);
      return typeof e3 == "symbol" ? e3 : String(e3);
    }(qe = "DEFAULT_PROGRAM_NAME")) in ze ? Object.defineProperty(ze, qe, {value: He, enumerable: true, configurable: true, writable: true}) : ze[qe] = He;
    class Ke {
      constructor(t3, e3, n2, s2) {
        this.tokens = t3, this.instructionIndex = n2, this.index = e3, this.text = s2 === void 0 ? "" : s2;
      }
      toString() {
        let t3 = this.constructor.name;
        const e3 = t3.indexOf("$");
        return t3 = t3.substring(e3 + 1, t3.length), "<" + t3 + "@" + this.tokens.get(this.index) + ':"' + this.text + '">';
      }
    }
    class Ye extends Ke {
      constructor(t3, e3, n2, s2) {
        super(t3, e3, n2, s2);
      }
      execute(e3) {
        return this.text && e3.push(this.text.toString()), this.tokens.get(this.index).type !== t2.EOF && e3.push(String(this.tokens.get(this.index).text)), this.index + 1;
      }
    }
    class Ge extends Ye {
      constructor(t3, e3, n2, s2) {
        super(t3, e3 + 1, n2, s2);
      }
    }
    class je extends Ke {
      constructor(t3, e3, n2, s2, i3) {
        super(t3, e3, s2, i3), this.lastIndex = n2;
      }
      execute(t3) {
        return this.text && t3.push(this.text.toString()), this.lastIndex + 1;
      }
      toString() {
        return this.text == null ? "<DeleteOp@" + this.tokens.get(this.index) + ".." + this.tokens.get(this.lastIndex) + ">" : "<ReplaceOp@" + this.tokens.get(this.index) + ".." + this.tokens.get(this.lastIndex) + ':"' + this.text + '">';
      }
    }
    const We = {atn: re, dfa: he, context: ce, misc: ue, tree: fe, error: Ne, Token: t2, CommonToken: Pt, CharStreams: ve, CharStream: Ie, InputStream: Ie, CommonTokenStream: be, Lexer: Ut, Parser: Fe, ParserRuleContext: Be, Interval: m2, IntervalSet: E2, LL1Analyzer: W, Utils: Re, TokenStreamRewriter: Ve};
  })();
  var i = s.dx;
  var r = s.q2;
  var o = s.FO;
  var a = s.xf;
  var l = s.Gy;
  var h = s.s4;
  var c = s.c7;
  var u = s._7;
  var d = s.tx;
  var p = s.gp;
  var g = s.cK;
  var f = s.zs;
  var x = s.AV;
  var T = s.Xp;
  var S = s.VS;
  var m = s.ul;
  var E = s.hW;
  var _ = s.x1;
  var A = s.z5;
  var C = s.oN;
  var N = s.TB;
  var y = s.u1;
  var I = s._b;
  var k = s.$F;
  var L = s._T;
  var O = s.db;
  var v = s.Zx;
  var R = s._x;
  var w = s.r8;
  var P = s.JI;
  var b = s.TP;
  var D = s.WU;
  var F = s.Nj;
  var M = s.ZP;

  // src/antlr/SolidityLexer.ts
  var _SolidityLexer = class extends E {
    constructor(input) {
      super(input);
      this._interp = new _(this, _SolidityLexer._ATN, _SolidityLexer.DecisionsToDFA, new O());
    }
    get grammarFileName() {
      return "Solidity.g4";
    }
    get literalNames() {
      return _SolidityLexer.literalNames;
    }
    get symbolicNames() {
      return _SolidityLexer.symbolicNames;
    }
    get ruleNames() {
      return _SolidityLexer.ruleNames;
    }
    get serializedATN() {
      return _SolidityLexer._serializedATN;
    }
    get channelNames() {
      return _SolidityLexer.channelNames;
    }
    get modeNames() {
      return _SolidityLexer.modeNames;
    }
    static get _ATN() {
      if (!_SolidityLexer.__ATN) {
        _SolidityLexer.__ATN = new r().deserialize(_SolidityLexer._serializedATN);
      }
      return _SolidityLexer.__ATN;
    }
  };
  var SolidityLexer = _SolidityLexer;
  SolidityLexer.T__0 = 1;
  SolidityLexer.T__1 = 2;
  SolidityLexer.T__2 = 3;
  SolidityLexer.T__3 = 4;
  SolidityLexer.T__4 = 5;
  SolidityLexer.T__5 = 6;
  SolidityLexer.T__6 = 7;
  SolidityLexer.T__7 = 8;
  SolidityLexer.T__8 = 9;
  SolidityLexer.T__9 = 10;
  SolidityLexer.T__10 = 11;
  SolidityLexer.T__11 = 12;
  SolidityLexer.T__12 = 13;
  SolidityLexer.T__13 = 14;
  SolidityLexer.T__14 = 15;
  SolidityLexer.T__15 = 16;
  SolidityLexer.T__16 = 17;
  SolidityLexer.T__17 = 18;
  SolidityLexer.T__18 = 19;
  SolidityLexer.T__19 = 20;
  SolidityLexer.T__20 = 21;
  SolidityLexer.T__21 = 22;
  SolidityLexer.T__22 = 23;
  SolidityLexer.T__23 = 24;
  SolidityLexer.T__24 = 25;
  SolidityLexer.T__25 = 26;
  SolidityLexer.T__26 = 27;
  SolidityLexer.T__27 = 28;
  SolidityLexer.T__28 = 29;
  SolidityLexer.T__29 = 30;
  SolidityLexer.T__30 = 31;
  SolidityLexer.T__31 = 32;
  SolidityLexer.T__32 = 33;
  SolidityLexer.T__33 = 34;
  SolidityLexer.T__34 = 35;
  SolidityLexer.T__35 = 36;
  SolidityLexer.T__36 = 37;
  SolidityLexer.T__37 = 38;
  SolidityLexer.T__38 = 39;
  SolidityLexer.T__39 = 40;
  SolidityLexer.T__40 = 41;
  SolidityLexer.T__41 = 42;
  SolidityLexer.T__42 = 43;
  SolidityLexer.T__43 = 44;
  SolidityLexer.T__44 = 45;
  SolidityLexer.T__45 = 46;
  SolidityLexer.T__46 = 47;
  SolidityLexer.T__47 = 48;
  SolidityLexer.T__48 = 49;
  SolidityLexer.T__49 = 50;
  SolidityLexer.T__50 = 51;
  SolidityLexer.T__51 = 52;
  SolidityLexer.T__52 = 53;
  SolidityLexer.T__53 = 54;
  SolidityLexer.T__54 = 55;
  SolidityLexer.T__55 = 56;
  SolidityLexer.T__56 = 57;
  SolidityLexer.T__57 = 58;
  SolidityLexer.T__58 = 59;
  SolidityLexer.T__59 = 60;
  SolidityLexer.T__60 = 61;
  SolidityLexer.T__61 = 62;
  SolidityLexer.T__62 = 63;
  SolidityLexer.T__63 = 64;
  SolidityLexer.T__64 = 65;
  SolidityLexer.T__65 = 66;
  SolidityLexer.T__66 = 67;
  SolidityLexer.T__67 = 68;
  SolidityLexer.T__68 = 69;
  SolidityLexer.T__69 = 70;
  SolidityLexer.T__70 = 71;
  SolidityLexer.T__71 = 72;
  SolidityLexer.T__72 = 73;
  SolidityLexer.T__73 = 74;
  SolidityLexer.T__74 = 75;
  SolidityLexer.T__75 = 76;
  SolidityLexer.T__76 = 77;
  SolidityLexer.T__77 = 78;
  SolidityLexer.T__78 = 79;
  SolidityLexer.T__79 = 80;
  SolidityLexer.T__80 = 81;
  SolidityLexer.T__81 = 82;
  SolidityLexer.T__82 = 83;
  SolidityLexer.T__83 = 84;
  SolidityLexer.T__84 = 85;
  SolidityLexer.T__85 = 86;
  SolidityLexer.T__86 = 87;
  SolidityLexer.T__87 = 88;
  SolidityLexer.T__88 = 89;
  SolidityLexer.T__89 = 90;
  SolidityLexer.T__90 = 91;
  SolidityLexer.T__91 = 92;
  SolidityLexer.T__92 = 93;
  SolidityLexer.T__93 = 94;
  SolidityLexer.T__94 = 95;
  SolidityLexer.T__95 = 96;
  SolidityLexer.Int = 97;
  SolidityLexer.Uint = 98;
  SolidityLexer.Byte = 99;
  SolidityLexer.Fixed = 100;
  SolidityLexer.Ufixed = 101;
  SolidityLexer.BooleanLiteral = 102;
  SolidityLexer.DecimalNumber = 103;
  SolidityLexer.HexNumber = 104;
  SolidityLexer.NumberUnit = 105;
  SolidityLexer.HexLiteralFragment = 106;
  SolidityLexer.ReservedKeyword = 107;
  SolidityLexer.AnonymousKeyword = 108;
  SolidityLexer.BreakKeyword = 109;
  SolidityLexer.ConstantKeyword = 110;
  SolidityLexer.ImmutableKeyword = 111;
  SolidityLexer.ContinueKeyword = 112;
  SolidityLexer.LeaveKeyword = 113;
  SolidityLexer.ExternalKeyword = 114;
  SolidityLexer.IndexedKeyword = 115;
  SolidityLexer.InternalKeyword = 116;
  SolidityLexer.PayableKeyword = 117;
  SolidityLexer.PrivateKeyword = 118;
  SolidityLexer.PublicKeyword = 119;
  SolidityLexer.VirtualKeyword = 120;
  SolidityLexer.PureKeyword = 121;
  SolidityLexer.TypeKeyword = 122;
  SolidityLexer.ViewKeyword = 123;
  SolidityLexer.GlobalKeyword = 124;
  SolidityLexer.ConstructorKeyword = 125;
  SolidityLexer.FallbackKeyword = 126;
  SolidityLexer.ReceiveKeyword = 127;
  SolidityLexer.Identifier = 128;
  SolidityLexer.StringLiteralFragment = 129;
  SolidityLexer.VersionLiteral = 130;
  SolidityLexer.WS = 131;
  SolidityLexer.COMMENT = 132;
  SolidityLexer.LINE_COMMENT = 133;
  SolidityLexer.EOF = D.EOF;
  SolidityLexer.channelNames = ["DEFAULT_TOKEN_CHANNEL", "HIDDEN"];
  SolidityLexer.literalNames = [
    null,
    "'pragma'",
    "';'",
    "'*'",
    "'||'",
    "'^'",
    "'~'",
    "'>='",
    "'>'",
    "'<'",
    "'<='",
    "'='",
    "'as'",
    "'import'",
    "'from'",
    "'{'",
    "','",
    "'}'",
    "'abstract'",
    "'contract'",
    "'interface'",
    "'library'",
    "'is'",
    "'('",
    "')'",
    "'error'",
    "'using'",
    "'for'",
    "'|'",
    "'&'",
    "'+'",
    "'-'",
    "'/'",
    "'%'",
    "'=='",
    "'!='",
    "'struct'",
    "'modifier'",
    "'function'",
    "'returns'",
    "'event'",
    "'enum'",
    "'['",
    "']'",
    "'address'",
    "'.'",
    "'mapping'",
    "'=>'",
    "'memory'",
    "'storage'",
    "'calldata'",
    "'if'",
    "'else'",
    "'try'",
    "'catch'",
    "'while'",
    "'unchecked'",
    "'assembly'",
    "'do'",
    "'return'",
    "'throw'",
    "'emit'",
    "'revert'",
    "'var'",
    "'bool'",
    "'string'",
    "'byte'",
    "'++'",
    "'--'",
    "'new'",
    "':'",
    "'delete'",
    "'!'",
    "'**'",
    "'<<'",
    "'>>'",
    "'&&'",
    "'?'",
    "'|='",
    "'^='",
    "'&='",
    "'<<='",
    "'>>='",
    "'+='",
    "'-='",
    "'*='",
    "'/='",
    "'%='",
    "'let'",
    "':='",
    "'=:'",
    "'switch'",
    "'case'",
    "'default'",
    "'->'",
    "'callback'",
    "'override'",
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    "'anonymous'",
    "'break'",
    "'constant'",
    "'immutable'",
    "'continue'",
    "'leave'",
    "'external'",
    "'indexed'",
    "'internal'",
    "'payable'",
    "'private'",
    "'public'",
    "'virtual'",
    "'pure'",
    "'type'",
    "'view'",
    "'global'",
    "'constructor'",
    "'fallback'",
    "'receive'"
  ];
  SolidityLexer.symbolicNames = [
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    "Int",
    "Uint",
    "Byte",
    "Fixed",
    "Ufixed",
    "BooleanLiteral",
    "DecimalNumber",
    "HexNumber",
    "NumberUnit",
    "HexLiteralFragment",
    "ReservedKeyword",
    "AnonymousKeyword",
    "BreakKeyword",
    "ConstantKeyword",
    "ImmutableKeyword",
    "ContinueKeyword",
    "LeaveKeyword",
    "ExternalKeyword",
    "IndexedKeyword",
    "InternalKeyword",
    "PayableKeyword",
    "PrivateKeyword",
    "PublicKeyword",
    "VirtualKeyword",
    "PureKeyword",
    "TypeKeyword",
    "ViewKeyword",
    "GlobalKeyword",
    "ConstructorKeyword",
    "FallbackKeyword",
    "ReceiveKeyword",
    "Identifier",
    "StringLiteralFragment",
    "VersionLiteral",
    "WS",
    "COMMENT",
    "LINE_COMMENT"
  ];
  SolidityLexer.modeNames = ["DEFAULT_MODE"];
  SolidityLexer.ruleNames = [
    "T__0",
    "T__1",
    "T__2",
    "T__3",
    "T__4",
    "T__5",
    "T__6",
    "T__7",
    "T__8",
    "T__9",
    "T__10",
    "T__11",
    "T__12",
    "T__13",
    "T__14",
    "T__15",
    "T__16",
    "T__17",
    "T__18",
    "T__19",
    "T__20",
    "T__21",
    "T__22",
    "T__23",
    "T__24",
    "T__25",
    "T__26",
    "T__27",
    "T__28",
    "T__29",
    "T__30",
    "T__31",
    "T__32",
    "T__33",
    "T__34",
    "T__35",
    "T__36",
    "T__37",
    "T__38",
    "T__39",
    "T__40",
    "T__41",
    "T__42",
    "T__43",
    "T__44",
    "T__45",
    "T__46",
    "T__47",
    "T__48",
    "T__49",
    "T__50",
    "T__51",
    "T__52",
    "T__53",
    "T__54",
    "T__55",
    "T__56",
    "T__57",
    "T__58",
    "T__59",
    "T__60",
    "T__61",
    "T__62",
    "T__63",
    "T__64",
    "T__65",
    "T__66",
    "T__67",
    "T__68",
    "T__69",
    "T__70",
    "T__71",
    "T__72",
    "T__73",
    "T__74",
    "T__75",
    "T__76",
    "T__77",
    "T__78",
    "T__79",
    "T__80",
    "T__81",
    "T__82",
    "T__83",
    "T__84",
    "T__85",
    "T__86",
    "T__87",
    "T__88",
    "T__89",
    "T__90",
    "T__91",
    "T__92",
    "T__93",
    "T__94",
    "T__95",
    "Int",
    "Uint",
    "Byte",
    "Fixed",
    "Ufixed",
    "NumberOfBits",
    "NumberOfBytes",
    "BooleanLiteral",
    "DecimalNumber",
    "DecimalDigits",
    "HexNumber",
    "HexDigits",
    "NumberUnit",
    "HexLiteralFragment",
    "HexCharacter",
    "ReservedKeyword",
    "AnonymousKeyword",
    "BreakKeyword",
    "ConstantKeyword",
    "ImmutableKeyword",
    "ContinueKeyword",
    "LeaveKeyword",
    "ExternalKeyword",
    "IndexedKeyword",
    "InternalKeyword",
    "PayableKeyword",
    "PrivateKeyword",
    "PublicKeyword",
    "VirtualKeyword",
    "PureKeyword",
    "TypeKeyword",
    "ViewKeyword",
    "GlobalKeyword",
    "ConstructorKeyword",
    "FallbackKeyword",
    "ReceiveKeyword",
    "Identifier",
    "IdentifierStart",
    "IdentifierPart",
    "StringLiteralFragment",
    "DoubleQuotedStringCharacter",
    "SingleQuotedStringCharacter",
    "VersionLiteral",
    "WS",
    "COMMENT",
    "LINE_COMMENT"
  ];
  SolidityLexer._serializedATN = [
    4,
    0,
    133,
    1336,
    6,
    -1,
    2,
    0,
    7,
    0,
    2,
    1,
    7,
    1,
    2,
    2,
    7,
    2,
    2,
    3,
    7,
    3,
    2,
    4,
    7,
    4,
    2,
    5,
    7,
    5,
    2,
    6,
    7,
    6,
    2,
    7,
    7,
    7,
    2,
    8,
    7,
    8,
    2,
    9,
    7,
    9,
    2,
    10,
    7,
    10,
    2,
    11,
    7,
    11,
    2,
    12,
    7,
    12,
    2,
    13,
    7,
    13,
    2,
    14,
    7,
    14,
    2,
    15,
    7,
    15,
    2,
    16,
    7,
    16,
    2,
    17,
    7,
    17,
    2,
    18,
    7,
    18,
    2,
    19,
    7,
    19,
    2,
    20,
    7,
    20,
    2,
    21,
    7,
    21,
    2,
    22,
    7,
    22,
    2,
    23,
    7,
    23,
    2,
    24,
    7,
    24,
    2,
    25,
    7,
    25,
    2,
    26,
    7,
    26,
    2,
    27,
    7,
    27,
    2,
    28,
    7,
    28,
    2,
    29,
    7,
    29,
    2,
    30,
    7,
    30,
    2,
    31,
    7,
    31,
    2,
    32,
    7,
    32,
    2,
    33,
    7,
    33,
    2,
    34,
    7,
    34,
    2,
    35,
    7,
    35,
    2,
    36,
    7,
    36,
    2,
    37,
    7,
    37,
    2,
    38,
    7,
    38,
    2,
    39,
    7,
    39,
    2,
    40,
    7,
    40,
    2,
    41,
    7,
    41,
    2,
    42,
    7,
    42,
    2,
    43,
    7,
    43,
    2,
    44,
    7,
    44,
    2,
    45,
    7,
    45,
    2,
    46,
    7,
    46,
    2,
    47,
    7,
    47,
    2,
    48,
    7,
    48,
    2,
    49,
    7,
    49,
    2,
    50,
    7,
    50,
    2,
    51,
    7,
    51,
    2,
    52,
    7,
    52,
    2,
    53,
    7,
    53,
    2,
    54,
    7,
    54,
    2,
    55,
    7,
    55,
    2,
    56,
    7,
    56,
    2,
    57,
    7,
    57,
    2,
    58,
    7,
    58,
    2,
    59,
    7,
    59,
    2,
    60,
    7,
    60,
    2,
    61,
    7,
    61,
    2,
    62,
    7,
    62,
    2,
    63,
    7,
    63,
    2,
    64,
    7,
    64,
    2,
    65,
    7,
    65,
    2,
    66,
    7,
    66,
    2,
    67,
    7,
    67,
    2,
    68,
    7,
    68,
    2,
    69,
    7,
    69,
    2,
    70,
    7,
    70,
    2,
    71,
    7,
    71,
    2,
    72,
    7,
    72,
    2,
    73,
    7,
    73,
    2,
    74,
    7,
    74,
    2,
    75,
    7,
    75,
    2,
    76,
    7,
    76,
    2,
    77,
    7,
    77,
    2,
    78,
    7,
    78,
    2,
    79,
    7,
    79,
    2,
    80,
    7,
    80,
    2,
    81,
    7,
    81,
    2,
    82,
    7,
    82,
    2,
    83,
    7,
    83,
    2,
    84,
    7,
    84,
    2,
    85,
    7,
    85,
    2,
    86,
    7,
    86,
    2,
    87,
    7,
    87,
    2,
    88,
    7,
    88,
    2,
    89,
    7,
    89,
    2,
    90,
    7,
    90,
    2,
    91,
    7,
    91,
    2,
    92,
    7,
    92,
    2,
    93,
    7,
    93,
    2,
    94,
    7,
    94,
    2,
    95,
    7,
    95,
    2,
    96,
    7,
    96,
    2,
    97,
    7,
    97,
    2,
    98,
    7,
    98,
    2,
    99,
    7,
    99,
    2,
    100,
    7,
    100,
    2,
    101,
    7,
    101,
    2,
    102,
    7,
    102,
    2,
    103,
    7,
    103,
    2,
    104,
    7,
    104,
    2,
    105,
    7,
    105,
    2,
    106,
    7,
    106,
    2,
    107,
    7,
    107,
    2,
    108,
    7,
    108,
    2,
    109,
    7,
    109,
    2,
    110,
    7,
    110,
    2,
    111,
    7,
    111,
    2,
    112,
    7,
    112,
    2,
    113,
    7,
    113,
    2,
    114,
    7,
    114,
    2,
    115,
    7,
    115,
    2,
    116,
    7,
    116,
    2,
    117,
    7,
    117,
    2,
    118,
    7,
    118,
    2,
    119,
    7,
    119,
    2,
    120,
    7,
    120,
    2,
    121,
    7,
    121,
    2,
    122,
    7,
    122,
    2,
    123,
    7,
    123,
    2,
    124,
    7,
    124,
    2,
    125,
    7,
    125,
    2,
    126,
    7,
    126,
    2,
    127,
    7,
    127,
    2,
    128,
    7,
    128,
    2,
    129,
    7,
    129,
    2,
    130,
    7,
    130,
    2,
    131,
    7,
    131,
    2,
    132,
    7,
    132,
    2,
    133,
    7,
    133,
    2,
    134,
    7,
    134,
    2,
    135,
    7,
    135,
    2,
    136,
    7,
    136,
    2,
    137,
    7,
    137,
    2,
    138,
    7,
    138,
    2,
    139,
    7,
    139,
    2,
    140,
    7,
    140,
    2,
    141,
    7,
    141,
    1,
    0,
    1,
    0,
    1,
    0,
    1,
    0,
    1,
    0,
    1,
    0,
    1,
    0,
    1,
    1,
    1,
    1,
    1,
    2,
    1,
    2,
    1,
    3,
    1,
    3,
    1,
    3,
    1,
    4,
    1,
    4,
    1,
    5,
    1,
    5,
    1,
    6,
    1,
    6,
    1,
    6,
    1,
    7,
    1,
    7,
    1,
    8,
    1,
    8,
    1,
    9,
    1,
    9,
    1,
    9,
    1,
    10,
    1,
    10,
    1,
    11,
    1,
    11,
    1,
    11,
    1,
    12,
    1,
    12,
    1,
    12,
    1,
    12,
    1,
    12,
    1,
    12,
    1,
    12,
    1,
    13,
    1,
    13,
    1,
    13,
    1,
    13,
    1,
    13,
    1,
    14,
    1,
    14,
    1,
    15,
    1,
    15,
    1,
    16,
    1,
    16,
    1,
    17,
    1,
    17,
    1,
    17,
    1,
    17,
    1,
    17,
    1,
    17,
    1,
    17,
    1,
    17,
    1,
    17,
    1,
    18,
    1,
    18,
    1,
    18,
    1,
    18,
    1,
    18,
    1,
    18,
    1,
    18,
    1,
    18,
    1,
    18,
    1,
    19,
    1,
    19,
    1,
    19,
    1,
    19,
    1,
    19,
    1,
    19,
    1,
    19,
    1,
    19,
    1,
    19,
    1,
    19,
    1,
    20,
    1,
    20,
    1,
    20,
    1,
    20,
    1,
    20,
    1,
    20,
    1,
    20,
    1,
    20,
    1,
    21,
    1,
    21,
    1,
    21,
    1,
    22,
    1,
    22,
    1,
    23,
    1,
    23,
    1,
    24,
    1,
    24,
    1,
    24,
    1,
    24,
    1,
    24,
    1,
    24,
    1,
    25,
    1,
    25,
    1,
    25,
    1,
    25,
    1,
    25,
    1,
    25,
    1,
    26,
    1,
    26,
    1,
    26,
    1,
    26,
    1,
    27,
    1,
    27,
    1,
    28,
    1,
    28,
    1,
    29,
    1,
    29,
    1,
    30,
    1,
    30,
    1,
    31,
    1,
    31,
    1,
    32,
    1,
    32,
    1,
    33,
    1,
    33,
    1,
    33,
    1,
    34,
    1,
    34,
    1,
    34,
    1,
    35,
    1,
    35,
    1,
    35,
    1,
    35,
    1,
    35,
    1,
    35,
    1,
    35,
    1,
    36,
    1,
    36,
    1,
    36,
    1,
    36,
    1,
    36,
    1,
    36,
    1,
    36,
    1,
    36,
    1,
    36,
    1,
    37,
    1,
    37,
    1,
    37,
    1,
    37,
    1,
    37,
    1,
    37,
    1,
    37,
    1,
    37,
    1,
    37,
    1,
    38,
    1,
    38,
    1,
    38,
    1,
    38,
    1,
    38,
    1,
    38,
    1,
    38,
    1,
    38,
    1,
    39,
    1,
    39,
    1,
    39,
    1,
    39,
    1,
    39,
    1,
    39,
    1,
    40,
    1,
    40,
    1,
    40,
    1,
    40,
    1,
    40,
    1,
    41,
    1,
    41,
    1,
    42,
    1,
    42,
    1,
    43,
    1,
    43,
    1,
    43,
    1,
    43,
    1,
    43,
    1,
    43,
    1,
    43,
    1,
    43,
    1,
    44,
    1,
    44,
    1,
    45,
    1,
    45,
    1,
    45,
    1,
    45,
    1,
    45,
    1,
    45,
    1,
    45,
    1,
    45,
    1,
    46,
    1,
    46,
    1,
    46,
    1,
    47,
    1,
    47,
    1,
    47,
    1,
    47,
    1,
    47,
    1,
    47,
    1,
    47,
    1,
    48,
    1,
    48,
    1,
    48,
    1,
    48,
    1,
    48,
    1,
    48,
    1,
    48,
    1,
    48,
    1,
    49,
    1,
    49,
    1,
    49,
    1,
    49,
    1,
    49,
    1,
    49,
    1,
    49,
    1,
    49,
    1,
    49,
    1,
    50,
    1,
    50,
    1,
    50,
    1,
    51,
    1,
    51,
    1,
    51,
    1,
    51,
    1,
    51,
    1,
    52,
    1,
    52,
    1,
    52,
    1,
    52,
    1,
    53,
    1,
    53,
    1,
    53,
    1,
    53,
    1,
    53,
    1,
    53,
    1,
    54,
    1,
    54,
    1,
    54,
    1,
    54,
    1,
    54,
    1,
    54,
    1,
    55,
    1,
    55,
    1,
    55,
    1,
    55,
    1,
    55,
    1,
    55,
    1,
    55,
    1,
    55,
    1,
    55,
    1,
    55,
    1,
    56,
    1,
    56,
    1,
    56,
    1,
    56,
    1,
    56,
    1,
    56,
    1,
    56,
    1,
    56,
    1,
    56,
    1,
    57,
    1,
    57,
    1,
    57,
    1,
    58,
    1,
    58,
    1,
    58,
    1,
    58,
    1,
    58,
    1,
    58,
    1,
    58,
    1,
    59,
    1,
    59,
    1,
    59,
    1,
    59,
    1,
    59,
    1,
    59,
    1,
    60,
    1,
    60,
    1,
    60,
    1,
    60,
    1,
    60,
    1,
    61,
    1,
    61,
    1,
    61,
    1,
    61,
    1,
    61,
    1,
    61,
    1,
    61,
    1,
    62,
    1,
    62,
    1,
    62,
    1,
    62,
    1,
    63,
    1,
    63,
    1,
    63,
    1,
    63,
    1,
    63,
    1,
    64,
    1,
    64,
    1,
    64,
    1,
    64,
    1,
    64,
    1,
    64,
    1,
    64,
    1,
    65,
    1,
    65,
    1,
    65,
    1,
    65,
    1,
    65,
    1,
    66,
    1,
    66,
    1,
    66,
    1,
    67,
    1,
    67,
    1,
    67,
    1,
    68,
    1,
    68,
    1,
    68,
    1,
    68,
    1,
    69,
    1,
    69,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    71,
    1,
    71,
    1,
    72,
    1,
    72,
    1,
    72,
    1,
    73,
    1,
    73,
    1,
    73,
    1,
    74,
    1,
    74,
    1,
    74,
    1,
    75,
    1,
    75,
    1,
    75,
    1,
    76,
    1,
    76,
    1,
    77,
    1,
    77,
    1,
    77,
    1,
    78,
    1,
    78,
    1,
    78,
    1,
    79,
    1,
    79,
    1,
    79,
    1,
    80,
    1,
    80,
    1,
    80,
    1,
    80,
    1,
    81,
    1,
    81,
    1,
    81,
    1,
    81,
    1,
    82,
    1,
    82,
    1,
    82,
    1,
    83,
    1,
    83,
    1,
    83,
    1,
    84,
    1,
    84,
    1,
    84,
    1,
    85,
    1,
    85,
    1,
    85,
    1,
    86,
    1,
    86,
    1,
    86,
    1,
    87,
    1,
    87,
    1,
    87,
    1,
    87,
    1,
    88,
    1,
    88,
    1,
    88,
    1,
    89,
    1,
    89,
    1,
    89,
    1,
    90,
    1,
    90,
    1,
    90,
    1,
    90,
    1,
    90,
    1,
    90,
    1,
    90,
    1,
    91,
    1,
    91,
    1,
    91,
    1,
    91,
    1,
    91,
    1,
    92,
    1,
    92,
    1,
    92,
    1,
    92,
    1,
    92,
    1,
    92,
    1,
    92,
    1,
    92,
    1,
    93,
    1,
    93,
    1,
    93,
    1,
    94,
    1,
    94,
    1,
    94,
    1,
    94,
    1,
    94,
    1,
    94,
    1,
    94,
    1,
    94,
    1,
    94,
    1,
    95,
    1,
    95,
    1,
    95,
    1,
    95,
    1,
    95,
    1,
    95,
    1,
    95,
    1,
    95,
    1,
    95,
    1,
    96,
    1,
    96,
    1,
    96,
    1,
    96,
    1,
    96,
    3,
    96,
    722,
    8,
    96,
    1,
    97,
    1,
    97,
    1,
    97,
    1,
    97,
    1,
    97,
    1,
    97,
    3,
    97,
    730,
    8,
    97,
    1,
    98,
    1,
    98,
    1,
    98,
    1,
    98,
    1,
    98,
    1,
    98,
    1,
    98,
    3,
    98,
    739,
    8,
    98,
    1,
    99,
    1,
    99,
    1,
    99,
    1,
    99,
    1,
    99,
    1,
    99,
    1,
    99,
    1,
    99,
    1,
    99,
    4,
    99,
    750,
    8,
    99,
    11,
    99,
    12,
    99,
    751,
    3,
    99,
    754,
    8,
    99,
    1,
    100,
    1,
    100,
    1,
    100,
    1,
    100,
    1,
    100,
    1,
    100,
    1,
    100,
    1,
    100,
    1,
    100,
    1,
    100,
    4,
    100,
    766,
    8,
    100,
    11,
    100,
    12,
    100,
    767,
    3,
    100,
    770,
    8,
    100,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    1,
    101,
    3,
    101,
    855,
    8,
    101,
    1,
    102,
    1,
    102,
    1,
    102,
    1,
    102,
    1,
    102,
    3,
    102,
    862,
    8,
    102,
    1,
    103,
    1,
    103,
    1,
    103,
    1,
    103,
    1,
    103,
    1,
    103,
    1,
    103,
    1,
    103,
    1,
    103,
    3,
    103,
    873,
    8,
    103,
    1,
    104,
    1,
    104,
    3,
    104,
    877,
    8,
    104,
    1,
    104,
    1,
    104,
    3,
    104,
    881,
    8,
    104,
    1,
    104,
    1,
    104,
    3,
    104,
    885,
    8,
    104,
    1,
    104,
    3,
    104,
    888,
    8,
    104,
    1,
    105,
    1,
    105,
    3,
    105,
    892,
    8,
    105,
    1,
    105,
    5,
    105,
    895,
    8,
    105,
    10,
    105,
    12,
    105,
    898,
    9,
    105,
    1,
    106,
    1,
    106,
    1,
    106,
    1,
    106,
    1,
    107,
    1,
    107,
    3,
    107,
    906,
    8,
    107,
    1,
    107,
    5,
    107,
    909,
    8,
    107,
    10,
    107,
    12,
    107,
    912,
    9,
    107,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    1,
    108,
    3,
    108,
    970,
    8,
    108,
    1,
    109,
    1,
    109,
    1,
    109,
    1,
    109,
    1,
    109,
    1,
    109,
    3,
    109,
    978,
    8,
    109,
    1,
    109,
    1,
    109,
    1,
    109,
    3,
    109,
    983,
    8,
    109,
    1,
    109,
    3,
    109,
    986,
    8,
    109,
    1,
    110,
    1,
    110,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    1,
    111,
    3,
    111,
    1078,
    8,
    111,
    1,
    112,
    1,
    112,
    1,
    112,
    1,
    112,
    1,
    112,
    1,
    112,
    1,
    112,
    1,
    112,
    1,
    112,
    1,
    112,
    1,
    113,
    1,
    113,
    1,
    113,
    1,
    113,
    1,
    113,
    1,
    113,
    1,
    114,
    1,
    114,
    1,
    114,
    1,
    114,
    1,
    114,
    1,
    114,
    1,
    114,
    1,
    114,
    1,
    114,
    1,
    115,
    1,
    115,
    1,
    115,
    1,
    115,
    1,
    115,
    1,
    115,
    1,
    115,
    1,
    115,
    1,
    115,
    1,
    115,
    1,
    116,
    1,
    116,
    1,
    116,
    1,
    116,
    1,
    116,
    1,
    116,
    1,
    116,
    1,
    116,
    1,
    116,
    1,
    117,
    1,
    117,
    1,
    117,
    1,
    117,
    1,
    117,
    1,
    117,
    1,
    118,
    1,
    118,
    1,
    118,
    1,
    118,
    1,
    118,
    1,
    118,
    1,
    118,
    1,
    118,
    1,
    118,
    1,
    119,
    1,
    119,
    1,
    119,
    1,
    119,
    1,
    119,
    1,
    119,
    1,
    119,
    1,
    119,
    1,
    120,
    1,
    120,
    1,
    120,
    1,
    120,
    1,
    120,
    1,
    120,
    1,
    120,
    1,
    120,
    1,
    120,
    1,
    121,
    1,
    121,
    1,
    121,
    1,
    121,
    1,
    121,
    1,
    121,
    1,
    121,
    1,
    121,
    1,
    122,
    1,
    122,
    1,
    122,
    1,
    122,
    1,
    122,
    1,
    122,
    1,
    122,
    1,
    122,
    1,
    123,
    1,
    123,
    1,
    123,
    1,
    123,
    1,
    123,
    1,
    123,
    1,
    123,
    1,
    124,
    1,
    124,
    1,
    124,
    1,
    124,
    1,
    124,
    1,
    124,
    1,
    124,
    1,
    124,
    1,
    125,
    1,
    125,
    1,
    125,
    1,
    125,
    1,
    125,
    1,
    126,
    1,
    126,
    1,
    126,
    1,
    126,
    1,
    126,
    1,
    127,
    1,
    127,
    1,
    127,
    1,
    127,
    1,
    127,
    1,
    128,
    1,
    128,
    1,
    128,
    1,
    128,
    1,
    128,
    1,
    128,
    1,
    128,
    1,
    129,
    1,
    129,
    1,
    129,
    1,
    129,
    1,
    129,
    1,
    129,
    1,
    129,
    1,
    129,
    1,
    129,
    1,
    129,
    1,
    129,
    1,
    129,
    1,
    130,
    1,
    130,
    1,
    130,
    1,
    130,
    1,
    130,
    1,
    130,
    1,
    130,
    1,
    130,
    1,
    130,
    1,
    131,
    1,
    131,
    1,
    131,
    1,
    131,
    1,
    131,
    1,
    131,
    1,
    131,
    1,
    131,
    1,
    132,
    1,
    132,
    5,
    132,
    1240,
    8,
    132,
    10,
    132,
    12,
    132,
    1243,
    9,
    132,
    1,
    133,
    1,
    133,
    1,
    134,
    1,
    134,
    1,
    135,
    1,
    135,
    1,
    135,
    1,
    135,
    1,
    135,
    1,
    135,
    1,
    135,
    3,
    135,
    1256,
    8,
    135,
    1,
    135,
    1,
    135,
    5,
    135,
    1260,
    8,
    135,
    10,
    135,
    12,
    135,
    1263,
    9,
    135,
    1,
    135,
    1,
    135,
    1,
    135,
    5,
    135,
    1268,
    8,
    135,
    10,
    135,
    12,
    135,
    1271,
    9,
    135,
    1,
    135,
    3,
    135,
    1274,
    8,
    135,
    1,
    136,
    1,
    136,
    1,
    136,
    3,
    136,
    1279,
    8,
    136,
    1,
    137,
    1,
    137,
    1,
    137,
    3,
    137,
    1284,
    8,
    137,
    1,
    138,
    4,
    138,
    1287,
    8,
    138,
    11,
    138,
    12,
    138,
    1288,
    1,
    138,
    1,
    138,
    4,
    138,
    1293,
    8,
    138,
    11,
    138,
    12,
    138,
    1294,
    1,
    138,
    1,
    138,
    4,
    138,
    1299,
    8,
    138,
    11,
    138,
    12,
    138,
    1300,
    3,
    138,
    1303,
    8,
    138,
    1,
    139,
    4,
    139,
    1306,
    8,
    139,
    11,
    139,
    12,
    139,
    1307,
    1,
    139,
    1,
    139,
    1,
    140,
    1,
    140,
    1,
    140,
    1,
    140,
    5,
    140,
    1316,
    8,
    140,
    10,
    140,
    12,
    140,
    1319,
    9,
    140,
    1,
    140,
    1,
    140,
    1,
    140,
    1,
    140,
    1,
    140,
    1,
    141,
    1,
    141,
    1,
    141,
    1,
    141,
    5,
    141,
    1330,
    8,
    141,
    10,
    141,
    12,
    141,
    1333,
    9,
    141,
    1,
    141,
    1,
    141,
    1,
    1317,
    0,
    142,
    1,
    1,
    3,
    2,
    5,
    3,
    7,
    4,
    9,
    5,
    11,
    6,
    13,
    7,
    15,
    8,
    17,
    9,
    19,
    10,
    21,
    11,
    23,
    12,
    25,
    13,
    27,
    14,
    29,
    15,
    31,
    16,
    33,
    17,
    35,
    18,
    37,
    19,
    39,
    20,
    41,
    21,
    43,
    22,
    45,
    23,
    47,
    24,
    49,
    25,
    51,
    26,
    53,
    27,
    55,
    28,
    57,
    29,
    59,
    30,
    61,
    31,
    63,
    32,
    65,
    33,
    67,
    34,
    69,
    35,
    71,
    36,
    73,
    37,
    75,
    38,
    77,
    39,
    79,
    40,
    81,
    41,
    83,
    42,
    85,
    43,
    87,
    44,
    89,
    45,
    91,
    46,
    93,
    47,
    95,
    48,
    97,
    49,
    99,
    50,
    101,
    51,
    103,
    52,
    105,
    53,
    107,
    54,
    109,
    55,
    111,
    56,
    113,
    57,
    115,
    58,
    117,
    59,
    119,
    60,
    121,
    61,
    123,
    62,
    125,
    63,
    127,
    64,
    129,
    65,
    131,
    66,
    133,
    67,
    135,
    68,
    137,
    69,
    139,
    70,
    141,
    71,
    143,
    72,
    145,
    73,
    147,
    74,
    149,
    75,
    151,
    76,
    153,
    77,
    155,
    78,
    157,
    79,
    159,
    80,
    161,
    81,
    163,
    82,
    165,
    83,
    167,
    84,
    169,
    85,
    171,
    86,
    173,
    87,
    175,
    88,
    177,
    89,
    179,
    90,
    181,
    91,
    183,
    92,
    185,
    93,
    187,
    94,
    189,
    95,
    191,
    96,
    193,
    97,
    195,
    98,
    197,
    99,
    199,
    100,
    201,
    101,
    203,
    0,
    205,
    0,
    207,
    102,
    209,
    103,
    211,
    0,
    213,
    104,
    215,
    0,
    217,
    105,
    219,
    106,
    221,
    0,
    223,
    107,
    225,
    108,
    227,
    109,
    229,
    110,
    231,
    111,
    233,
    112,
    235,
    113,
    237,
    114,
    239,
    115,
    241,
    116,
    243,
    117,
    245,
    118,
    247,
    119,
    249,
    120,
    251,
    121,
    253,
    122,
    255,
    123,
    257,
    124,
    259,
    125,
    261,
    126,
    263,
    127,
    265,
    128,
    267,
    0,
    269,
    0,
    271,
    129,
    273,
    0,
    275,
    0,
    277,
    130,
    279,
    131,
    281,
    132,
    283,
    133,
    1,
    0,
    13,
    1,
    0,
    48,
    57,
    1,
    0,
    49,
    57,
    1,
    0,
    49,
    50,
    1,
    0,
    48,
    50,
    2,
    0,
    69,
    69,
    101,
    101,
    2,
    0,
    88,
    88,
    120,
    120,
    3,
    0,
    48,
    57,
    65,
    70,
    97,
    102,
    4,
    0,
    36,
    36,
    65,
    90,
    95,
    95,
    97,
    122,
    5,
    0,
    36,
    36,
    48,
    57,
    65,
    90,
    95,
    95,
    97,
    122,
    4,
    0,
    10,
    10,
    13,
    13,
    34,
    34,
    92,
    92,
    4,
    0,
    10,
    10,
    13,
    13,
    39,
    39,
    92,
    92,
    3,
    0,
    9,
    10,
    12,
    13,
    32,
    32,
    2,
    0,
    10,
    10,
    13,
    13,
    1418,
    0,
    1,
    1,
    0,
    0,
    0,
    0,
    3,
    1,
    0,
    0,
    0,
    0,
    5,
    1,
    0,
    0,
    0,
    0,
    7,
    1,
    0,
    0,
    0,
    0,
    9,
    1,
    0,
    0,
    0,
    0,
    11,
    1,
    0,
    0,
    0,
    0,
    13,
    1,
    0,
    0,
    0,
    0,
    15,
    1,
    0,
    0,
    0,
    0,
    17,
    1,
    0,
    0,
    0,
    0,
    19,
    1,
    0,
    0,
    0,
    0,
    21,
    1,
    0,
    0,
    0,
    0,
    23,
    1,
    0,
    0,
    0,
    0,
    25,
    1,
    0,
    0,
    0,
    0,
    27,
    1,
    0,
    0,
    0,
    0,
    29,
    1,
    0,
    0,
    0,
    0,
    31,
    1,
    0,
    0,
    0,
    0,
    33,
    1,
    0,
    0,
    0,
    0,
    35,
    1,
    0,
    0,
    0,
    0,
    37,
    1,
    0,
    0,
    0,
    0,
    39,
    1,
    0,
    0,
    0,
    0,
    41,
    1,
    0,
    0,
    0,
    0,
    43,
    1,
    0,
    0,
    0,
    0,
    45,
    1,
    0,
    0,
    0,
    0,
    47,
    1,
    0,
    0,
    0,
    0,
    49,
    1,
    0,
    0,
    0,
    0,
    51,
    1,
    0,
    0,
    0,
    0,
    53,
    1,
    0,
    0,
    0,
    0,
    55,
    1,
    0,
    0,
    0,
    0,
    57,
    1,
    0,
    0,
    0,
    0,
    59,
    1,
    0,
    0,
    0,
    0,
    61,
    1,
    0,
    0,
    0,
    0,
    63,
    1,
    0,
    0,
    0,
    0,
    65,
    1,
    0,
    0,
    0,
    0,
    67,
    1,
    0,
    0,
    0,
    0,
    69,
    1,
    0,
    0,
    0,
    0,
    71,
    1,
    0,
    0,
    0,
    0,
    73,
    1,
    0,
    0,
    0,
    0,
    75,
    1,
    0,
    0,
    0,
    0,
    77,
    1,
    0,
    0,
    0,
    0,
    79,
    1,
    0,
    0,
    0,
    0,
    81,
    1,
    0,
    0,
    0,
    0,
    83,
    1,
    0,
    0,
    0,
    0,
    85,
    1,
    0,
    0,
    0,
    0,
    87,
    1,
    0,
    0,
    0,
    0,
    89,
    1,
    0,
    0,
    0,
    0,
    91,
    1,
    0,
    0,
    0,
    0,
    93,
    1,
    0,
    0,
    0,
    0,
    95,
    1,
    0,
    0,
    0,
    0,
    97,
    1,
    0,
    0,
    0,
    0,
    99,
    1,
    0,
    0,
    0,
    0,
    101,
    1,
    0,
    0,
    0,
    0,
    103,
    1,
    0,
    0,
    0,
    0,
    105,
    1,
    0,
    0,
    0,
    0,
    107,
    1,
    0,
    0,
    0,
    0,
    109,
    1,
    0,
    0,
    0,
    0,
    111,
    1,
    0,
    0,
    0,
    0,
    113,
    1,
    0,
    0,
    0,
    0,
    115,
    1,
    0,
    0,
    0,
    0,
    117,
    1,
    0,
    0,
    0,
    0,
    119,
    1,
    0,
    0,
    0,
    0,
    121,
    1,
    0,
    0,
    0,
    0,
    123,
    1,
    0,
    0,
    0,
    0,
    125,
    1,
    0,
    0,
    0,
    0,
    127,
    1,
    0,
    0,
    0,
    0,
    129,
    1,
    0,
    0,
    0,
    0,
    131,
    1,
    0,
    0,
    0,
    0,
    133,
    1,
    0,
    0,
    0,
    0,
    135,
    1,
    0,
    0,
    0,
    0,
    137,
    1,
    0,
    0,
    0,
    0,
    139,
    1,
    0,
    0,
    0,
    0,
    141,
    1,
    0,
    0,
    0,
    0,
    143,
    1,
    0,
    0,
    0,
    0,
    145,
    1,
    0,
    0,
    0,
    0,
    147,
    1,
    0,
    0,
    0,
    0,
    149,
    1,
    0,
    0,
    0,
    0,
    151,
    1,
    0,
    0,
    0,
    0,
    153,
    1,
    0,
    0,
    0,
    0,
    155,
    1,
    0,
    0,
    0,
    0,
    157,
    1,
    0,
    0,
    0,
    0,
    159,
    1,
    0,
    0,
    0,
    0,
    161,
    1,
    0,
    0,
    0,
    0,
    163,
    1,
    0,
    0,
    0,
    0,
    165,
    1,
    0,
    0,
    0,
    0,
    167,
    1,
    0,
    0,
    0,
    0,
    169,
    1,
    0,
    0,
    0,
    0,
    171,
    1,
    0,
    0,
    0,
    0,
    173,
    1,
    0,
    0,
    0,
    0,
    175,
    1,
    0,
    0,
    0,
    0,
    177,
    1,
    0,
    0,
    0,
    0,
    179,
    1,
    0,
    0,
    0,
    0,
    181,
    1,
    0,
    0,
    0,
    0,
    183,
    1,
    0,
    0,
    0,
    0,
    185,
    1,
    0,
    0,
    0,
    0,
    187,
    1,
    0,
    0,
    0,
    0,
    189,
    1,
    0,
    0,
    0,
    0,
    191,
    1,
    0,
    0,
    0,
    0,
    193,
    1,
    0,
    0,
    0,
    0,
    195,
    1,
    0,
    0,
    0,
    0,
    197,
    1,
    0,
    0,
    0,
    0,
    199,
    1,
    0,
    0,
    0,
    0,
    201,
    1,
    0,
    0,
    0,
    0,
    207,
    1,
    0,
    0,
    0,
    0,
    209,
    1,
    0,
    0,
    0,
    0,
    213,
    1,
    0,
    0,
    0,
    0,
    217,
    1,
    0,
    0,
    0,
    0,
    219,
    1,
    0,
    0,
    0,
    0,
    223,
    1,
    0,
    0,
    0,
    0,
    225,
    1,
    0,
    0,
    0,
    0,
    227,
    1,
    0,
    0,
    0,
    0,
    229,
    1,
    0,
    0,
    0,
    0,
    231,
    1,
    0,
    0,
    0,
    0,
    233,
    1,
    0,
    0,
    0,
    0,
    235,
    1,
    0,
    0,
    0,
    0,
    237,
    1,
    0,
    0,
    0,
    0,
    239,
    1,
    0,
    0,
    0,
    0,
    241,
    1,
    0,
    0,
    0,
    0,
    243,
    1,
    0,
    0,
    0,
    0,
    245,
    1,
    0,
    0,
    0,
    0,
    247,
    1,
    0,
    0,
    0,
    0,
    249,
    1,
    0,
    0,
    0,
    0,
    251,
    1,
    0,
    0,
    0,
    0,
    253,
    1,
    0,
    0,
    0,
    0,
    255,
    1,
    0,
    0,
    0,
    0,
    257,
    1,
    0,
    0,
    0,
    0,
    259,
    1,
    0,
    0,
    0,
    0,
    261,
    1,
    0,
    0,
    0,
    0,
    263,
    1,
    0,
    0,
    0,
    0,
    265,
    1,
    0,
    0,
    0,
    0,
    271,
    1,
    0,
    0,
    0,
    0,
    277,
    1,
    0,
    0,
    0,
    0,
    279,
    1,
    0,
    0,
    0,
    0,
    281,
    1,
    0,
    0,
    0,
    0,
    283,
    1,
    0,
    0,
    0,
    1,
    285,
    1,
    0,
    0,
    0,
    3,
    292,
    1,
    0,
    0,
    0,
    5,
    294,
    1,
    0,
    0,
    0,
    7,
    296,
    1,
    0,
    0,
    0,
    9,
    299,
    1,
    0,
    0,
    0,
    11,
    301,
    1,
    0,
    0,
    0,
    13,
    303,
    1,
    0,
    0,
    0,
    15,
    306,
    1,
    0,
    0,
    0,
    17,
    308,
    1,
    0,
    0,
    0,
    19,
    310,
    1,
    0,
    0,
    0,
    21,
    313,
    1,
    0,
    0,
    0,
    23,
    315,
    1,
    0,
    0,
    0,
    25,
    318,
    1,
    0,
    0,
    0,
    27,
    325,
    1,
    0,
    0,
    0,
    29,
    330,
    1,
    0,
    0,
    0,
    31,
    332,
    1,
    0,
    0,
    0,
    33,
    334,
    1,
    0,
    0,
    0,
    35,
    336,
    1,
    0,
    0,
    0,
    37,
    345,
    1,
    0,
    0,
    0,
    39,
    354,
    1,
    0,
    0,
    0,
    41,
    364,
    1,
    0,
    0,
    0,
    43,
    372,
    1,
    0,
    0,
    0,
    45,
    375,
    1,
    0,
    0,
    0,
    47,
    377,
    1,
    0,
    0,
    0,
    49,
    379,
    1,
    0,
    0,
    0,
    51,
    385,
    1,
    0,
    0,
    0,
    53,
    391,
    1,
    0,
    0,
    0,
    55,
    395,
    1,
    0,
    0,
    0,
    57,
    397,
    1,
    0,
    0,
    0,
    59,
    399,
    1,
    0,
    0,
    0,
    61,
    401,
    1,
    0,
    0,
    0,
    63,
    403,
    1,
    0,
    0,
    0,
    65,
    405,
    1,
    0,
    0,
    0,
    67,
    407,
    1,
    0,
    0,
    0,
    69,
    410,
    1,
    0,
    0,
    0,
    71,
    413,
    1,
    0,
    0,
    0,
    73,
    420,
    1,
    0,
    0,
    0,
    75,
    429,
    1,
    0,
    0,
    0,
    77,
    438,
    1,
    0,
    0,
    0,
    79,
    446,
    1,
    0,
    0,
    0,
    81,
    452,
    1,
    0,
    0,
    0,
    83,
    457,
    1,
    0,
    0,
    0,
    85,
    459,
    1,
    0,
    0,
    0,
    87,
    461,
    1,
    0,
    0,
    0,
    89,
    469,
    1,
    0,
    0,
    0,
    91,
    471,
    1,
    0,
    0,
    0,
    93,
    479,
    1,
    0,
    0,
    0,
    95,
    482,
    1,
    0,
    0,
    0,
    97,
    489,
    1,
    0,
    0,
    0,
    99,
    497,
    1,
    0,
    0,
    0,
    101,
    506,
    1,
    0,
    0,
    0,
    103,
    509,
    1,
    0,
    0,
    0,
    105,
    514,
    1,
    0,
    0,
    0,
    107,
    518,
    1,
    0,
    0,
    0,
    109,
    524,
    1,
    0,
    0,
    0,
    111,
    530,
    1,
    0,
    0,
    0,
    113,
    540,
    1,
    0,
    0,
    0,
    115,
    549,
    1,
    0,
    0,
    0,
    117,
    552,
    1,
    0,
    0,
    0,
    119,
    559,
    1,
    0,
    0,
    0,
    121,
    565,
    1,
    0,
    0,
    0,
    123,
    570,
    1,
    0,
    0,
    0,
    125,
    577,
    1,
    0,
    0,
    0,
    127,
    581,
    1,
    0,
    0,
    0,
    129,
    586,
    1,
    0,
    0,
    0,
    131,
    593,
    1,
    0,
    0,
    0,
    133,
    598,
    1,
    0,
    0,
    0,
    135,
    601,
    1,
    0,
    0,
    0,
    137,
    604,
    1,
    0,
    0,
    0,
    139,
    608,
    1,
    0,
    0,
    0,
    141,
    610,
    1,
    0,
    0,
    0,
    143,
    617,
    1,
    0,
    0,
    0,
    145,
    619,
    1,
    0,
    0,
    0,
    147,
    622,
    1,
    0,
    0,
    0,
    149,
    625,
    1,
    0,
    0,
    0,
    151,
    628,
    1,
    0,
    0,
    0,
    153,
    631,
    1,
    0,
    0,
    0,
    155,
    633,
    1,
    0,
    0,
    0,
    157,
    636,
    1,
    0,
    0,
    0,
    159,
    639,
    1,
    0,
    0,
    0,
    161,
    642,
    1,
    0,
    0,
    0,
    163,
    646,
    1,
    0,
    0,
    0,
    165,
    650,
    1,
    0,
    0,
    0,
    167,
    653,
    1,
    0,
    0,
    0,
    169,
    656,
    1,
    0,
    0,
    0,
    171,
    659,
    1,
    0,
    0,
    0,
    173,
    662,
    1,
    0,
    0,
    0,
    175,
    665,
    1,
    0,
    0,
    0,
    177,
    669,
    1,
    0,
    0,
    0,
    179,
    672,
    1,
    0,
    0,
    0,
    181,
    675,
    1,
    0,
    0,
    0,
    183,
    682,
    1,
    0,
    0,
    0,
    185,
    687,
    1,
    0,
    0,
    0,
    187,
    695,
    1,
    0,
    0,
    0,
    189,
    698,
    1,
    0,
    0,
    0,
    191,
    707,
    1,
    0,
    0,
    0,
    193,
    716,
    1,
    0,
    0,
    0,
    195,
    723,
    1,
    0,
    0,
    0,
    197,
    731,
    1,
    0,
    0,
    0,
    199,
    740,
    1,
    0,
    0,
    0,
    201,
    755,
    1,
    0,
    0,
    0,
    203,
    854,
    1,
    0,
    0,
    0,
    205,
    861,
    1,
    0,
    0,
    0,
    207,
    872,
    1,
    0,
    0,
    0,
    209,
    880,
    1,
    0,
    0,
    0,
    211,
    889,
    1,
    0,
    0,
    0,
    213,
    899,
    1,
    0,
    0,
    0,
    215,
    903,
    1,
    0,
    0,
    0,
    217,
    969,
    1,
    0,
    0,
    0,
    219,
    971,
    1,
    0,
    0,
    0,
    221,
    987,
    1,
    0,
    0,
    0,
    223,
    1077,
    1,
    0,
    0,
    0,
    225,
    1079,
    1,
    0,
    0,
    0,
    227,
    1089,
    1,
    0,
    0,
    0,
    229,
    1095,
    1,
    0,
    0,
    0,
    231,
    1104,
    1,
    0,
    0,
    0,
    233,
    1114,
    1,
    0,
    0,
    0,
    235,
    1123,
    1,
    0,
    0,
    0,
    237,
    1129,
    1,
    0,
    0,
    0,
    239,
    1138,
    1,
    0,
    0,
    0,
    241,
    1146,
    1,
    0,
    0,
    0,
    243,
    1155,
    1,
    0,
    0,
    0,
    245,
    1163,
    1,
    0,
    0,
    0,
    247,
    1171,
    1,
    0,
    0,
    0,
    249,
    1178,
    1,
    0,
    0,
    0,
    251,
    1186,
    1,
    0,
    0,
    0,
    253,
    1191,
    1,
    0,
    0,
    0,
    255,
    1196,
    1,
    0,
    0,
    0,
    257,
    1201,
    1,
    0,
    0,
    0,
    259,
    1208,
    1,
    0,
    0,
    0,
    261,
    1220,
    1,
    0,
    0,
    0,
    263,
    1229,
    1,
    0,
    0,
    0,
    265,
    1237,
    1,
    0,
    0,
    0,
    267,
    1244,
    1,
    0,
    0,
    0,
    269,
    1246,
    1,
    0,
    0,
    0,
    271,
    1255,
    1,
    0,
    0,
    0,
    273,
    1278,
    1,
    0,
    0,
    0,
    275,
    1283,
    1,
    0,
    0,
    0,
    277,
    1286,
    1,
    0,
    0,
    0,
    279,
    1305,
    1,
    0,
    0,
    0,
    281,
    1311,
    1,
    0,
    0,
    0,
    283,
    1325,
    1,
    0,
    0,
    0,
    285,
    286,
    5,
    112,
    0,
    0,
    286,
    287,
    5,
    114,
    0,
    0,
    287,
    288,
    5,
    97,
    0,
    0,
    288,
    289,
    5,
    103,
    0,
    0,
    289,
    290,
    5,
    109,
    0,
    0,
    290,
    291,
    5,
    97,
    0,
    0,
    291,
    2,
    1,
    0,
    0,
    0,
    292,
    293,
    5,
    59,
    0,
    0,
    293,
    4,
    1,
    0,
    0,
    0,
    294,
    295,
    5,
    42,
    0,
    0,
    295,
    6,
    1,
    0,
    0,
    0,
    296,
    297,
    5,
    124,
    0,
    0,
    297,
    298,
    5,
    124,
    0,
    0,
    298,
    8,
    1,
    0,
    0,
    0,
    299,
    300,
    5,
    94,
    0,
    0,
    300,
    10,
    1,
    0,
    0,
    0,
    301,
    302,
    5,
    126,
    0,
    0,
    302,
    12,
    1,
    0,
    0,
    0,
    303,
    304,
    5,
    62,
    0,
    0,
    304,
    305,
    5,
    61,
    0,
    0,
    305,
    14,
    1,
    0,
    0,
    0,
    306,
    307,
    5,
    62,
    0,
    0,
    307,
    16,
    1,
    0,
    0,
    0,
    308,
    309,
    5,
    60,
    0,
    0,
    309,
    18,
    1,
    0,
    0,
    0,
    310,
    311,
    5,
    60,
    0,
    0,
    311,
    312,
    5,
    61,
    0,
    0,
    312,
    20,
    1,
    0,
    0,
    0,
    313,
    314,
    5,
    61,
    0,
    0,
    314,
    22,
    1,
    0,
    0,
    0,
    315,
    316,
    5,
    97,
    0,
    0,
    316,
    317,
    5,
    115,
    0,
    0,
    317,
    24,
    1,
    0,
    0,
    0,
    318,
    319,
    5,
    105,
    0,
    0,
    319,
    320,
    5,
    109,
    0,
    0,
    320,
    321,
    5,
    112,
    0,
    0,
    321,
    322,
    5,
    111,
    0,
    0,
    322,
    323,
    5,
    114,
    0,
    0,
    323,
    324,
    5,
    116,
    0,
    0,
    324,
    26,
    1,
    0,
    0,
    0,
    325,
    326,
    5,
    102,
    0,
    0,
    326,
    327,
    5,
    114,
    0,
    0,
    327,
    328,
    5,
    111,
    0,
    0,
    328,
    329,
    5,
    109,
    0,
    0,
    329,
    28,
    1,
    0,
    0,
    0,
    330,
    331,
    5,
    123,
    0,
    0,
    331,
    30,
    1,
    0,
    0,
    0,
    332,
    333,
    5,
    44,
    0,
    0,
    333,
    32,
    1,
    0,
    0,
    0,
    334,
    335,
    5,
    125,
    0,
    0,
    335,
    34,
    1,
    0,
    0,
    0,
    336,
    337,
    5,
    97,
    0,
    0,
    337,
    338,
    5,
    98,
    0,
    0,
    338,
    339,
    5,
    115,
    0,
    0,
    339,
    340,
    5,
    116,
    0,
    0,
    340,
    341,
    5,
    114,
    0,
    0,
    341,
    342,
    5,
    97,
    0,
    0,
    342,
    343,
    5,
    99,
    0,
    0,
    343,
    344,
    5,
    116,
    0,
    0,
    344,
    36,
    1,
    0,
    0,
    0,
    345,
    346,
    5,
    99,
    0,
    0,
    346,
    347,
    5,
    111,
    0,
    0,
    347,
    348,
    5,
    110,
    0,
    0,
    348,
    349,
    5,
    116,
    0,
    0,
    349,
    350,
    5,
    114,
    0,
    0,
    350,
    351,
    5,
    97,
    0,
    0,
    351,
    352,
    5,
    99,
    0,
    0,
    352,
    353,
    5,
    116,
    0,
    0,
    353,
    38,
    1,
    0,
    0,
    0,
    354,
    355,
    5,
    105,
    0,
    0,
    355,
    356,
    5,
    110,
    0,
    0,
    356,
    357,
    5,
    116,
    0,
    0,
    357,
    358,
    5,
    101,
    0,
    0,
    358,
    359,
    5,
    114,
    0,
    0,
    359,
    360,
    5,
    102,
    0,
    0,
    360,
    361,
    5,
    97,
    0,
    0,
    361,
    362,
    5,
    99,
    0,
    0,
    362,
    363,
    5,
    101,
    0,
    0,
    363,
    40,
    1,
    0,
    0,
    0,
    364,
    365,
    5,
    108,
    0,
    0,
    365,
    366,
    5,
    105,
    0,
    0,
    366,
    367,
    5,
    98,
    0,
    0,
    367,
    368,
    5,
    114,
    0,
    0,
    368,
    369,
    5,
    97,
    0,
    0,
    369,
    370,
    5,
    114,
    0,
    0,
    370,
    371,
    5,
    121,
    0,
    0,
    371,
    42,
    1,
    0,
    0,
    0,
    372,
    373,
    5,
    105,
    0,
    0,
    373,
    374,
    5,
    115,
    0,
    0,
    374,
    44,
    1,
    0,
    0,
    0,
    375,
    376,
    5,
    40,
    0,
    0,
    376,
    46,
    1,
    0,
    0,
    0,
    377,
    378,
    5,
    41,
    0,
    0,
    378,
    48,
    1,
    0,
    0,
    0,
    379,
    380,
    5,
    101,
    0,
    0,
    380,
    381,
    5,
    114,
    0,
    0,
    381,
    382,
    5,
    114,
    0,
    0,
    382,
    383,
    5,
    111,
    0,
    0,
    383,
    384,
    5,
    114,
    0,
    0,
    384,
    50,
    1,
    0,
    0,
    0,
    385,
    386,
    5,
    117,
    0,
    0,
    386,
    387,
    5,
    115,
    0,
    0,
    387,
    388,
    5,
    105,
    0,
    0,
    388,
    389,
    5,
    110,
    0,
    0,
    389,
    390,
    5,
    103,
    0,
    0,
    390,
    52,
    1,
    0,
    0,
    0,
    391,
    392,
    5,
    102,
    0,
    0,
    392,
    393,
    5,
    111,
    0,
    0,
    393,
    394,
    5,
    114,
    0,
    0,
    394,
    54,
    1,
    0,
    0,
    0,
    395,
    396,
    5,
    124,
    0,
    0,
    396,
    56,
    1,
    0,
    0,
    0,
    397,
    398,
    5,
    38,
    0,
    0,
    398,
    58,
    1,
    0,
    0,
    0,
    399,
    400,
    5,
    43,
    0,
    0,
    400,
    60,
    1,
    0,
    0,
    0,
    401,
    402,
    5,
    45,
    0,
    0,
    402,
    62,
    1,
    0,
    0,
    0,
    403,
    404,
    5,
    47,
    0,
    0,
    404,
    64,
    1,
    0,
    0,
    0,
    405,
    406,
    5,
    37,
    0,
    0,
    406,
    66,
    1,
    0,
    0,
    0,
    407,
    408,
    5,
    61,
    0,
    0,
    408,
    409,
    5,
    61,
    0,
    0,
    409,
    68,
    1,
    0,
    0,
    0,
    410,
    411,
    5,
    33,
    0,
    0,
    411,
    412,
    5,
    61,
    0,
    0,
    412,
    70,
    1,
    0,
    0,
    0,
    413,
    414,
    5,
    115,
    0,
    0,
    414,
    415,
    5,
    116,
    0,
    0,
    415,
    416,
    5,
    114,
    0,
    0,
    416,
    417,
    5,
    117,
    0,
    0,
    417,
    418,
    5,
    99,
    0,
    0,
    418,
    419,
    5,
    116,
    0,
    0,
    419,
    72,
    1,
    0,
    0,
    0,
    420,
    421,
    5,
    109,
    0,
    0,
    421,
    422,
    5,
    111,
    0,
    0,
    422,
    423,
    5,
    100,
    0,
    0,
    423,
    424,
    5,
    105,
    0,
    0,
    424,
    425,
    5,
    102,
    0,
    0,
    425,
    426,
    5,
    105,
    0,
    0,
    426,
    427,
    5,
    101,
    0,
    0,
    427,
    428,
    5,
    114,
    0,
    0,
    428,
    74,
    1,
    0,
    0,
    0,
    429,
    430,
    5,
    102,
    0,
    0,
    430,
    431,
    5,
    117,
    0,
    0,
    431,
    432,
    5,
    110,
    0,
    0,
    432,
    433,
    5,
    99,
    0,
    0,
    433,
    434,
    5,
    116,
    0,
    0,
    434,
    435,
    5,
    105,
    0,
    0,
    435,
    436,
    5,
    111,
    0,
    0,
    436,
    437,
    5,
    110,
    0,
    0,
    437,
    76,
    1,
    0,
    0,
    0,
    438,
    439,
    5,
    114,
    0,
    0,
    439,
    440,
    5,
    101,
    0,
    0,
    440,
    441,
    5,
    116,
    0,
    0,
    441,
    442,
    5,
    117,
    0,
    0,
    442,
    443,
    5,
    114,
    0,
    0,
    443,
    444,
    5,
    110,
    0,
    0,
    444,
    445,
    5,
    115,
    0,
    0,
    445,
    78,
    1,
    0,
    0,
    0,
    446,
    447,
    5,
    101,
    0,
    0,
    447,
    448,
    5,
    118,
    0,
    0,
    448,
    449,
    5,
    101,
    0,
    0,
    449,
    450,
    5,
    110,
    0,
    0,
    450,
    451,
    5,
    116,
    0,
    0,
    451,
    80,
    1,
    0,
    0,
    0,
    452,
    453,
    5,
    101,
    0,
    0,
    453,
    454,
    5,
    110,
    0,
    0,
    454,
    455,
    5,
    117,
    0,
    0,
    455,
    456,
    5,
    109,
    0,
    0,
    456,
    82,
    1,
    0,
    0,
    0,
    457,
    458,
    5,
    91,
    0,
    0,
    458,
    84,
    1,
    0,
    0,
    0,
    459,
    460,
    5,
    93,
    0,
    0,
    460,
    86,
    1,
    0,
    0,
    0,
    461,
    462,
    5,
    97,
    0,
    0,
    462,
    463,
    5,
    100,
    0,
    0,
    463,
    464,
    5,
    100,
    0,
    0,
    464,
    465,
    5,
    114,
    0,
    0,
    465,
    466,
    5,
    101,
    0,
    0,
    466,
    467,
    5,
    115,
    0,
    0,
    467,
    468,
    5,
    115,
    0,
    0,
    468,
    88,
    1,
    0,
    0,
    0,
    469,
    470,
    5,
    46,
    0,
    0,
    470,
    90,
    1,
    0,
    0,
    0,
    471,
    472,
    5,
    109,
    0,
    0,
    472,
    473,
    5,
    97,
    0,
    0,
    473,
    474,
    5,
    112,
    0,
    0,
    474,
    475,
    5,
    112,
    0,
    0,
    475,
    476,
    5,
    105,
    0,
    0,
    476,
    477,
    5,
    110,
    0,
    0,
    477,
    478,
    5,
    103,
    0,
    0,
    478,
    92,
    1,
    0,
    0,
    0,
    479,
    480,
    5,
    61,
    0,
    0,
    480,
    481,
    5,
    62,
    0,
    0,
    481,
    94,
    1,
    0,
    0,
    0,
    482,
    483,
    5,
    109,
    0,
    0,
    483,
    484,
    5,
    101,
    0,
    0,
    484,
    485,
    5,
    109,
    0,
    0,
    485,
    486,
    5,
    111,
    0,
    0,
    486,
    487,
    5,
    114,
    0,
    0,
    487,
    488,
    5,
    121,
    0,
    0,
    488,
    96,
    1,
    0,
    0,
    0,
    489,
    490,
    5,
    115,
    0,
    0,
    490,
    491,
    5,
    116,
    0,
    0,
    491,
    492,
    5,
    111,
    0,
    0,
    492,
    493,
    5,
    114,
    0,
    0,
    493,
    494,
    5,
    97,
    0,
    0,
    494,
    495,
    5,
    103,
    0,
    0,
    495,
    496,
    5,
    101,
    0,
    0,
    496,
    98,
    1,
    0,
    0,
    0,
    497,
    498,
    5,
    99,
    0,
    0,
    498,
    499,
    5,
    97,
    0,
    0,
    499,
    500,
    5,
    108,
    0,
    0,
    500,
    501,
    5,
    108,
    0,
    0,
    501,
    502,
    5,
    100,
    0,
    0,
    502,
    503,
    5,
    97,
    0,
    0,
    503,
    504,
    5,
    116,
    0,
    0,
    504,
    505,
    5,
    97,
    0,
    0,
    505,
    100,
    1,
    0,
    0,
    0,
    506,
    507,
    5,
    105,
    0,
    0,
    507,
    508,
    5,
    102,
    0,
    0,
    508,
    102,
    1,
    0,
    0,
    0,
    509,
    510,
    5,
    101,
    0,
    0,
    510,
    511,
    5,
    108,
    0,
    0,
    511,
    512,
    5,
    115,
    0,
    0,
    512,
    513,
    5,
    101,
    0,
    0,
    513,
    104,
    1,
    0,
    0,
    0,
    514,
    515,
    5,
    116,
    0,
    0,
    515,
    516,
    5,
    114,
    0,
    0,
    516,
    517,
    5,
    121,
    0,
    0,
    517,
    106,
    1,
    0,
    0,
    0,
    518,
    519,
    5,
    99,
    0,
    0,
    519,
    520,
    5,
    97,
    0,
    0,
    520,
    521,
    5,
    116,
    0,
    0,
    521,
    522,
    5,
    99,
    0,
    0,
    522,
    523,
    5,
    104,
    0,
    0,
    523,
    108,
    1,
    0,
    0,
    0,
    524,
    525,
    5,
    119,
    0,
    0,
    525,
    526,
    5,
    104,
    0,
    0,
    526,
    527,
    5,
    105,
    0,
    0,
    527,
    528,
    5,
    108,
    0,
    0,
    528,
    529,
    5,
    101,
    0,
    0,
    529,
    110,
    1,
    0,
    0,
    0,
    530,
    531,
    5,
    117,
    0,
    0,
    531,
    532,
    5,
    110,
    0,
    0,
    532,
    533,
    5,
    99,
    0,
    0,
    533,
    534,
    5,
    104,
    0,
    0,
    534,
    535,
    5,
    101,
    0,
    0,
    535,
    536,
    5,
    99,
    0,
    0,
    536,
    537,
    5,
    107,
    0,
    0,
    537,
    538,
    5,
    101,
    0,
    0,
    538,
    539,
    5,
    100,
    0,
    0,
    539,
    112,
    1,
    0,
    0,
    0,
    540,
    541,
    5,
    97,
    0,
    0,
    541,
    542,
    5,
    115,
    0,
    0,
    542,
    543,
    5,
    115,
    0,
    0,
    543,
    544,
    5,
    101,
    0,
    0,
    544,
    545,
    5,
    109,
    0,
    0,
    545,
    546,
    5,
    98,
    0,
    0,
    546,
    547,
    5,
    108,
    0,
    0,
    547,
    548,
    5,
    121,
    0,
    0,
    548,
    114,
    1,
    0,
    0,
    0,
    549,
    550,
    5,
    100,
    0,
    0,
    550,
    551,
    5,
    111,
    0,
    0,
    551,
    116,
    1,
    0,
    0,
    0,
    552,
    553,
    5,
    114,
    0,
    0,
    553,
    554,
    5,
    101,
    0,
    0,
    554,
    555,
    5,
    116,
    0,
    0,
    555,
    556,
    5,
    117,
    0,
    0,
    556,
    557,
    5,
    114,
    0,
    0,
    557,
    558,
    5,
    110,
    0,
    0,
    558,
    118,
    1,
    0,
    0,
    0,
    559,
    560,
    5,
    116,
    0,
    0,
    560,
    561,
    5,
    104,
    0,
    0,
    561,
    562,
    5,
    114,
    0,
    0,
    562,
    563,
    5,
    111,
    0,
    0,
    563,
    564,
    5,
    119,
    0,
    0,
    564,
    120,
    1,
    0,
    0,
    0,
    565,
    566,
    5,
    101,
    0,
    0,
    566,
    567,
    5,
    109,
    0,
    0,
    567,
    568,
    5,
    105,
    0,
    0,
    568,
    569,
    5,
    116,
    0,
    0,
    569,
    122,
    1,
    0,
    0,
    0,
    570,
    571,
    5,
    114,
    0,
    0,
    571,
    572,
    5,
    101,
    0,
    0,
    572,
    573,
    5,
    118,
    0,
    0,
    573,
    574,
    5,
    101,
    0,
    0,
    574,
    575,
    5,
    114,
    0,
    0,
    575,
    576,
    5,
    116,
    0,
    0,
    576,
    124,
    1,
    0,
    0,
    0,
    577,
    578,
    5,
    118,
    0,
    0,
    578,
    579,
    5,
    97,
    0,
    0,
    579,
    580,
    5,
    114,
    0,
    0,
    580,
    126,
    1,
    0,
    0,
    0,
    581,
    582,
    5,
    98,
    0,
    0,
    582,
    583,
    5,
    111,
    0,
    0,
    583,
    584,
    5,
    111,
    0,
    0,
    584,
    585,
    5,
    108,
    0,
    0,
    585,
    128,
    1,
    0,
    0,
    0,
    586,
    587,
    5,
    115,
    0,
    0,
    587,
    588,
    5,
    116,
    0,
    0,
    588,
    589,
    5,
    114,
    0,
    0,
    589,
    590,
    5,
    105,
    0,
    0,
    590,
    591,
    5,
    110,
    0,
    0,
    591,
    592,
    5,
    103,
    0,
    0,
    592,
    130,
    1,
    0,
    0,
    0,
    593,
    594,
    5,
    98,
    0,
    0,
    594,
    595,
    5,
    121,
    0,
    0,
    595,
    596,
    5,
    116,
    0,
    0,
    596,
    597,
    5,
    101,
    0,
    0,
    597,
    132,
    1,
    0,
    0,
    0,
    598,
    599,
    5,
    43,
    0,
    0,
    599,
    600,
    5,
    43,
    0,
    0,
    600,
    134,
    1,
    0,
    0,
    0,
    601,
    602,
    5,
    45,
    0,
    0,
    602,
    603,
    5,
    45,
    0,
    0,
    603,
    136,
    1,
    0,
    0,
    0,
    604,
    605,
    5,
    110,
    0,
    0,
    605,
    606,
    5,
    101,
    0,
    0,
    606,
    607,
    5,
    119,
    0,
    0,
    607,
    138,
    1,
    0,
    0,
    0,
    608,
    609,
    5,
    58,
    0,
    0,
    609,
    140,
    1,
    0,
    0,
    0,
    610,
    611,
    5,
    100,
    0,
    0,
    611,
    612,
    5,
    101,
    0,
    0,
    612,
    613,
    5,
    108,
    0,
    0,
    613,
    614,
    5,
    101,
    0,
    0,
    614,
    615,
    5,
    116,
    0,
    0,
    615,
    616,
    5,
    101,
    0,
    0,
    616,
    142,
    1,
    0,
    0,
    0,
    617,
    618,
    5,
    33,
    0,
    0,
    618,
    144,
    1,
    0,
    0,
    0,
    619,
    620,
    5,
    42,
    0,
    0,
    620,
    621,
    5,
    42,
    0,
    0,
    621,
    146,
    1,
    0,
    0,
    0,
    622,
    623,
    5,
    60,
    0,
    0,
    623,
    624,
    5,
    60,
    0,
    0,
    624,
    148,
    1,
    0,
    0,
    0,
    625,
    626,
    5,
    62,
    0,
    0,
    626,
    627,
    5,
    62,
    0,
    0,
    627,
    150,
    1,
    0,
    0,
    0,
    628,
    629,
    5,
    38,
    0,
    0,
    629,
    630,
    5,
    38,
    0,
    0,
    630,
    152,
    1,
    0,
    0,
    0,
    631,
    632,
    5,
    63,
    0,
    0,
    632,
    154,
    1,
    0,
    0,
    0,
    633,
    634,
    5,
    124,
    0,
    0,
    634,
    635,
    5,
    61,
    0,
    0,
    635,
    156,
    1,
    0,
    0,
    0,
    636,
    637,
    5,
    94,
    0,
    0,
    637,
    638,
    5,
    61,
    0,
    0,
    638,
    158,
    1,
    0,
    0,
    0,
    639,
    640,
    5,
    38,
    0,
    0,
    640,
    641,
    5,
    61,
    0,
    0,
    641,
    160,
    1,
    0,
    0,
    0,
    642,
    643,
    5,
    60,
    0,
    0,
    643,
    644,
    5,
    60,
    0,
    0,
    644,
    645,
    5,
    61,
    0,
    0,
    645,
    162,
    1,
    0,
    0,
    0,
    646,
    647,
    5,
    62,
    0,
    0,
    647,
    648,
    5,
    62,
    0,
    0,
    648,
    649,
    5,
    61,
    0,
    0,
    649,
    164,
    1,
    0,
    0,
    0,
    650,
    651,
    5,
    43,
    0,
    0,
    651,
    652,
    5,
    61,
    0,
    0,
    652,
    166,
    1,
    0,
    0,
    0,
    653,
    654,
    5,
    45,
    0,
    0,
    654,
    655,
    5,
    61,
    0,
    0,
    655,
    168,
    1,
    0,
    0,
    0,
    656,
    657,
    5,
    42,
    0,
    0,
    657,
    658,
    5,
    61,
    0,
    0,
    658,
    170,
    1,
    0,
    0,
    0,
    659,
    660,
    5,
    47,
    0,
    0,
    660,
    661,
    5,
    61,
    0,
    0,
    661,
    172,
    1,
    0,
    0,
    0,
    662,
    663,
    5,
    37,
    0,
    0,
    663,
    664,
    5,
    61,
    0,
    0,
    664,
    174,
    1,
    0,
    0,
    0,
    665,
    666,
    5,
    108,
    0,
    0,
    666,
    667,
    5,
    101,
    0,
    0,
    667,
    668,
    5,
    116,
    0,
    0,
    668,
    176,
    1,
    0,
    0,
    0,
    669,
    670,
    5,
    58,
    0,
    0,
    670,
    671,
    5,
    61,
    0,
    0,
    671,
    178,
    1,
    0,
    0,
    0,
    672,
    673,
    5,
    61,
    0,
    0,
    673,
    674,
    5,
    58,
    0,
    0,
    674,
    180,
    1,
    0,
    0,
    0,
    675,
    676,
    5,
    115,
    0,
    0,
    676,
    677,
    5,
    119,
    0,
    0,
    677,
    678,
    5,
    105,
    0,
    0,
    678,
    679,
    5,
    116,
    0,
    0,
    679,
    680,
    5,
    99,
    0,
    0,
    680,
    681,
    5,
    104,
    0,
    0,
    681,
    182,
    1,
    0,
    0,
    0,
    682,
    683,
    5,
    99,
    0,
    0,
    683,
    684,
    5,
    97,
    0,
    0,
    684,
    685,
    5,
    115,
    0,
    0,
    685,
    686,
    5,
    101,
    0,
    0,
    686,
    184,
    1,
    0,
    0,
    0,
    687,
    688,
    5,
    100,
    0,
    0,
    688,
    689,
    5,
    101,
    0,
    0,
    689,
    690,
    5,
    102,
    0,
    0,
    690,
    691,
    5,
    97,
    0,
    0,
    691,
    692,
    5,
    117,
    0,
    0,
    692,
    693,
    5,
    108,
    0,
    0,
    693,
    694,
    5,
    116,
    0,
    0,
    694,
    186,
    1,
    0,
    0,
    0,
    695,
    696,
    5,
    45,
    0,
    0,
    696,
    697,
    5,
    62,
    0,
    0,
    697,
    188,
    1,
    0,
    0,
    0,
    698,
    699,
    5,
    99,
    0,
    0,
    699,
    700,
    5,
    97,
    0,
    0,
    700,
    701,
    5,
    108,
    0,
    0,
    701,
    702,
    5,
    108,
    0,
    0,
    702,
    703,
    5,
    98,
    0,
    0,
    703,
    704,
    5,
    97,
    0,
    0,
    704,
    705,
    5,
    99,
    0,
    0,
    705,
    706,
    5,
    107,
    0,
    0,
    706,
    190,
    1,
    0,
    0,
    0,
    707,
    708,
    5,
    111,
    0,
    0,
    708,
    709,
    5,
    118,
    0,
    0,
    709,
    710,
    5,
    101,
    0,
    0,
    710,
    711,
    5,
    114,
    0,
    0,
    711,
    712,
    5,
    114,
    0,
    0,
    712,
    713,
    5,
    105,
    0,
    0,
    713,
    714,
    5,
    100,
    0,
    0,
    714,
    715,
    5,
    101,
    0,
    0,
    715,
    192,
    1,
    0,
    0,
    0,
    716,
    717,
    5,
    105,
    0,
    0,
    717,
    718,
    5,
    110,
    0,
    0,
    718,
    719,
    5,
    116,
    0,
    0,
    719,
    721,
    1,
    0,
    0,
    0,
    720,
    722,
    3,
    203,
    101,
    0,
    721,
    720,
    1,
    0,
    0,
    0,
    721,
    722,
    1,
    0,
    0,
    0,
    722,
    194,
    1,
    0,
    0,
    0,
    723,
    724,
    5,
    117,
    0,
    0,
    724,
    725,
    5,
    105,
    0,
    0,
    725,
    726,
    5,
    110,
    0,
    0,
    726,
    727,
    5,
    116,
    0,
    0,
    727,
    729,
    1,
    0,
    0,
    0,
    728,
    730,
    3,
    203,
    101,
    0,
    729,
    728,
    1,
    0,
    0,
    0,
    729,
    730,
    1,
    0,
    0,
    0,
    730,
    196,
    1,
    0,
    0,
    0,
    731,
    732,
    5,
    98,
    0,
    0,
    732,
    733,
    5,
    121,
    0,
    0,
    733,
    734,
    5,
    116,
    0,
    0,
    734,
    735,
    5,
    101,
    0,
    0,
    735,
    736,
    5,
    115,
    0,
    0,
    736,
    738,
    1,
    0,
    0,
    0,
    737,
    739,
    3,
    205,
    102,
    0,
    738,
    737,
    1,
    0,
    0,
    0,
    738,
    739,
    1,
    0,
    0,
    0,
    739,
    198,
    1,
    0,
    0,
    0,
    740,
    741,
    5,
    102,
    0,
    0,
    741,
    742,
    5,
    105,
    0,
    0,
    742,
    743,
    5,
    120,
    0,
    0,
    743,
    744,
    5,
    101,
    0,
    0,
    744,
    745,
    5,
    100,
    0,
    0,
    745,
    753,
    1,
    0,
    0,
    0,
    746,
    747,
    3,
    203,
    101,
    0,
    747,
    749,
    5,
    120,
    0,
    0,
    748,
    750,
    7,
    0,
    0,
    0,
    749,
    748,
    1,
    0,
    0,
    0,
    750,
    751,
    1,
    0,
    0,
    0,
    751,
    749,
    1,
    0,
    0,
    0,
    751,
    752,
    1,
    0,
    0,
    0,
    752,
    754,
    1,
    0,
    0,
    0,
    753,
    746,
    1,
    0,
    0,
    0,
    753,
    754,
    1,
    0,
    0,
    0,
    754,
    200,
    1,
    0,
    0,
    0,
    755,
    756,
    5,
    117,
    0,
    0,
    756,
    757,
    5,
    102,
    0,
    0,
    757,
    758,
    5,
    105,
    0,
    0,
    758,
    759,
    5,
    120,
    0,
    0,
    759,
    760,
    5,
    101,
    0,
    0,
    760,
    761,
    5,
    100,
    0,
    0,
    761,
    769,
    1,
    0,
    0,
    0,
    762,
    763,
    3,
    203,
    101,
    0,
    763,
    765,
    5,
    120,
    0,
    0,
    764,
    766,
    7,
    0,
    0,
    0,
    765,
    764,
    1,
    0,
    0,
    0,
    766,
    767,
    1,
    0,
    0,
    0,
    767,
    765,
    1,
    0,
    0,
    0,
    767,
    768,
    1,
    0,
    0,
    0,
    768,
    770,
    1,
    0,
    0,
    0,
    769,
    762,
    1,
    0,
    0,
    0,
    769,
    770,
    1,
    0,
    0,
    0,
    770,
    202,
    1,
    0,
    0,
    0,
    771,
    855,
    5,
    56,
    0,
    0,
    772,
    773,
    5,
    49,
    0,
    0,
    773,
    855,
    5,
    54,
    0,
    0,
    774,
    775,
    5,
    50,
    0,
    0,
    775,
    855,
    5,
    52,
    0,
    0,
    776,
    777,
    5,
    51,
    0,
    0,
    777,
    855,
    5,
    50,
    0,
    0,
    778,
    779,
    5,
    52,
    0,
    0,
    779,
    855,
    5,
    48,
    0,
    0,
    780,
    781,
    5,
    52,
    0,
    0,
    781,
    855,
    5,
    56,
    0,
    0,
    782,
    783,
    5,
    53,
    0,
    0,
    783,
    855,
    5,
    54,
    0,
    0,
    784,
    785,
    5,
    54,
    0,
    0,
    785,
    855,
    5,
    52,
    0,
    0,
    786,
    787,
    5,
    55,
    0,
    0,
    787,
    855,
    5,
    50,
    0,
    0,
    788,
    789,
    5,
    56,
    0,
    0,
    789,
    855,
    5,
    48,
    0,
    0,
    790,
    791,
    5,
    56,
    0,
    0,
    791,
    855,
    5,
    56,
    0,
    0,
    792,
    793,
    5,
    57,
    0,
    0,
    793,
    855,
    5,
    54,
    0,
    0,
    794,
    795,
    5,
    49,
    0,
    0,
    795,
    796,
    5,
    48,
    0,
    0,
    796,
    855,
    5,
    52,
    0,
    0,
    797,
    798,
    5,
    49,
    0,
    0,
    798,
    799,
    5,
    49,
    0,
    0,
    799,
    855,
    5,
    50,
    0,
    0,
    800,
    801,
    5,
    49,
    0,
    0,
    801,
    802,
    5,
    50,
    0,
    0,
    802,
    855,
    5,
    48,
    0,
    0,
    803,
    804,
    5,
    49,
    0,
    0,
    804,
    805,
    5,
    50,
    0,
    0,
    805,
    855,
    5,
    56,
    0,
    0,
    806,
    807,
    5,
    49,
    0,
    0,
    807,
    808,
    5,
    51,
    0,
    0,
    808,
    855,
    5,
    54,
    0,
    0,
    809,
    810,
    5,
    49,
    0,
    0,
    810,
    811,
    5,
    52,
    0,
    0,
    811,
    855,
    5,
    52,
    0,
    0,
    812,
    813,
    5,
    49,
    0,
    0,
    813,
    814,
    5,
    53,
    0,
    0,
    814,
    855,
    5,
    50,
    0,
    0,
    815,
    816,
    5,
    49,
    0,
    0,
    816,
    817,
    5,
    54,
    0,
    0,
    817,
    855,
    5,
    48,
    0,
    0,
    818,
    819,
    5,
    49,
    0,
    0,
    819,
    820,
    5,
    54,
    0,
    0,
    820,
    855,
    5,
    56,
    0,
    0,
    821,
    822,
    5,
    49,
    0,
    0,
    822,
    823,
    5,
    55,
    0,
    0,
    823,
    855,
    5,
    54,
    0,
    0,
    824,
    825,
    5,
    49,
    0,
    0,
    825,
    826,
    5,
    56,
    0,
    0,
    826,
    855,
    5,
    52,
    0,
    0,
    827,
    828,
    5,
    49,
    0,
    0,
    828,
    829,
    5,
    57,
    0,
    0,
    829,
    855,
    5,
    50,
    0,
    0,
    830,
    831,
    5,
    50,
    0,
    0,
    831,
    832,
    5,
    48,
    0,
    0,
    832,
    855,
    5,
    48,
    0,
    0,
    833,
    834,
    5,
    50,
    0,
    0,
    834,
    835,
    5,
    48,
    0,
    0,
    835,
    855,
    5,
    56,
    0,
    0,
    836,
    837,
    5,
    50,
    0,
    0,
    837,
    838,
    5,
    49,
    0,
    0,
    838,
    855,
    5,
    54,
    0,
    0,
    839,
    840,
    5,
    50,
    0,
    0,
    840,
    841,
    5,
    50,
    0,
    0,
    841,
    855,
    5,
    52,
    0,
    0,
    842,
    843,
    5,
    50,
    0,
    0,
    843,
    844,
    5,
    51,
    0,
    0,
    844,
    855,
    5,
    50,
    0,
    0,
    845,
    846,
    5,
    50,
    0,
    0,
    846,
    847,
    5,
    52,
    0,
    0,
    847,
    855,
    5,
    48,
    0,
    0,
    848,
    849,
    5,
    50,
    0,
    0,
    849,
    850,
    5,
    52,
    0,
    0,
    850,
    855,
    5,
    56,
    0,
    0,
    851,
    852,
    5,
    50,
    0,
    0,
    852,
    853,
    5,
    53,
    0,
    0,
    853,
    855,
    5,
    54,
    0,
    0,
    854,
    771,
    1,
    0,
    0,
    0,
    854,
    772,
    1,
    0,
    0,
    0,
    854,
    774,
    1,
    0,
    0,
    0,
    854,
    776,
    1,
    0,
    0,
    0,
    854,
    778,
    1,
    0,
    0,
    0,
    854,
    780,
    1,
    0,
    0,
    0,
    854,
    782,
    1,
    0,
    0,
    0,
    854,
    784,
    1,
    0,
    0,
    0,
    854,
    786,
    1,
    0,
    0,
    0,
    854,
    788,
    1,
    0,
    0,
    0,
    854,
    790,
    1,
    0,
    0,
    0,
    854,
    792,
    1,
    0,
    0,
    0,
    854,
    794,
    1,
    0,
    0,
    0,
    854,
    797,
    1,
    0,
    0,
    0,
    854,
    800,
    1,
    0,
    0,
    0,
    854,
    803,
    1,
    0,
    0,
    0,
    854,
    806,
    1,
    0,
    0,
    0,
    854,
    809,
    1,
    0,
    0,
    0,
    854,
    812,
    1,
    0,
    0,
    0,
    854,
    815,
    1,
    0,
    0,
    0,
    854,
    818,
    1,
    0,
    0,
    0,
    854,
    821,
    1,
    0,
    0,
    0,
    854,
    824,
    1,
    0,
    0,
    0,
    854,
    827,
    1,
    0,
    0,
    0,
    854,
    830,
    1,
    0,
    0,
    0,
    854,
    833,
    1,
    0,
    0,
    0,
    854,
    836,
    1,
    0,
    0,
    0,
    854,
    839,
    1,
    0,
    0,
    0,
    854,
    842,
    1,
    0,
    0,
    0,
    854,
    845,
    1,
    0,
    0,
    0,
    854,
    848,
    1,
    0,
    0,
    0,
    854,
    851,
    1,
    0,
    0,
    0,
    855,
    204,
    1,
    0,
    0,
    0,
    856,
    862,
    7,
    1,
    0,
    0,
    857,
    858,
    7,
    2,
    0,
    0,
    858,
    862,
    7,
    0,
    0,
    0,
    859,
    860,
    5,
    51,
    0,
    0,
    860,
    862,
    7,
    3,
    0,
    0,
    861,
    856,
    1,
    0,
    0,
    0,
    861,
    857,
    1,
    0,
    0,
    0,
    861,
    859,
    1,
    0,
    0,
    0,
    862,
    206,
    1,
    0,
    0,
    0,
    863,
    864,
    5,
    116,
    0,
    0,
    864,
    865,
    5,
    114,
    0,
    0,
    865,
    866,
    5,
    117,
    0,
    0,
    866,
    873,
    5,
    101,
    0,
    0,
    867,
    868,
    5,
    102,
    0,
    0,
    868,
    869,
    5,
    97,
    0,
    0,
    869,
    870,
    5,
    108,
    0,
    0,
    870,
    871,
    5,
    115,
    0,
    0,
    871,
    873,
    5,
    101,
    0,
    0,
    872,
    863,
    1,
    0,
    0,
    0,
    872,
    867,
    1,
    0,
    0,
    0,
    873,
    208,
    1,
    0,
    0,
    0,
    874,
    881,
    3,
    211,
    105,
    0,
    875,
    877,
    3,
    211,
    105,
    0,
    876,
    875,
    1,
    0,
    0,
    0,
    876,
    877,
    1,
    0,
    0,
    0,
    877,
    878,
    1,
    0,
    0,
    0,
    878,
    879,
    5,
    46,
    0,
    0,
    879,
    881,
    3,
    211,
    105,
    0,
    880,
    874,
    1,
    0,
    0,
    0,
    880,
    876,
    1,
    0,
    0,
    0,
    881,
    887,
    1,
    0,
    0,
    0,
    882,
    884,
    7,
    4,
    0,
    0,
    883,
    885,
    5,
    45,
    0,
    0,
    884,
    883,
    1,
    0,
    0,
    0,
    884,
    885,
    1,
    0,
    0,
    0,
    885,
    886,
    1,
    0,
    0,
    0,
    886,
    888,
    3,
    211,
    105,
    0,
    887,
    882,
    1,
    0,
    0,
    0,
    887,
    888,
    1,
    0,
    0,
    0,
    888,
    210,
    1,
    0,
    0,
    0,
    889,
    896,
    7,
    0,
    0,
    0,
    890,
    892,
    5,
    95,
    0,
    0,
    891,
    890,
    1,
    0,
    0,
    0,
    891,
    892,
    1,
    0,
    0,
    0,
    892,
    893,
    1,
    0,
    0,
    0,
    893,
    895,
    7,
    0,
    0,
    0,
    894,
    891,
    1,
    0,
    0,
    0,
    895,
    898,
    1,
    0,
    0,
    0,
    896,
    894,
    1,
    0,
    0,
    0,
    896,
    897,
    1,
    0,
    0,
    0,
    897,
    212,
    1,
    0,
    0,
    0,
    898,
    896,
    1,
    0,
    0,
    0,
    899,
    900,
    5,
    48,
    0,
    0,
    900,
    901,
    7,
    5,
    0,
    0,
    901,
    902,
    3,
    215,
    107,
    0,
    902,
    214,
    1,
    0,
    0,
    0,
    903,
    910,
    3,
    221,
    110,
    0,
    904,
    906,
    5,
    95,
    0,
    0,
    905,
    904,
    1,
    0,
    0,
    0,
    905,
    906,
    1,
    0,
    0,
    0,
    906,
    907,
    1,
    0,
    0,
    0,
    907,
    909,
    3,
    221,
    110,
    0,
    908,
    905,
    1,
    0,
    0,
    0,
    909,
    912,
    1,
    0,
    0,
    0,
    910,
    908,
    1,
    0,
    0,
    0,
    910,
    911,
    1,
    0,
    0,
    0,
    911,
    216,
    1,
    0,
    0,
    0,
    912,
    910,
    1,
    0,
    0,
    0,
    913,
    914,
    5,
    119,
    0,
    0,
    914,
    915,
    5,
    101,
    0,
    0,
    915,
    970,
    5,
    105,
    0,
    0,
    916,
    917,
    5,
    103,
    0,
    0,
    917,
    918,
    5,
    119,
    0,
    0,
    918,
    919,
    5,
    101,
    0,
    0,
    919,
    970,
    5,
    105,
    0,
    0,
    920,
    921,
    5,
    115,
    0,
    0,
    921,
    922,
    5,
    122,
    0,
    0,
    922,
    923,
    5,
    97,
    0,
    0,
    923,
    924,
    5,
    98,
    0,
    0,
    924,
    970,
    5,
    111,
    0,
    0,
    925,
    926,
    5,
    102,
    0,
    0,
    926,
    927,
    5,
    105,
    0,
    0,
    927,
    928,
    5,
    110,
    0,
    0,
    928,
    929,
    5,
    110,
    0,
    0,
    929,
    930,
    5,
    101,
    0,
    0,
    930,
    970,
    5,
    121,
    0,
    0,
    931,
    932,
    5,
    101,
    0,
    0,
    932,
    933,
    5,
    116,
    0,
    0,
    933,
    934,
    5,
    104,
    0,
    0,
    934,
    935,
    5,
    101,
    0,
    0,
    935,
    970,
    5,
    114,
    0,
    0,
    936,
    937,
    5,
    115,
    0,
    0,
    937,
    938,
    5,
    101,
    0,
    0,
    938,
    939,
    5,
    99,
    0,
    0,
    939,
    940,
    5,
    111,
    0,
    0,
    940,
    941,
    5,
    110,
    0,
    0,
    941,
    942,
    5,
    100,
    0,
    0,
    942,
    970,
    5,
    115,
    0,
    0,
    943,
    944,
    5,
    109,
    0,
    0,
    944,
    945,
    5,
    105,
    0,
    0,
    945,
    946,
    5,
    110,
    0,
    0,
    946,
    947,
    5,
    117,
    0,
    0,
    947,
    948,
    5,
    116,
    0,
    0,
    948,
    949,
    5,
    101,
    0,
    0,
    949,
    970,
    5,
    115,
    0,
    0,
    950,
    951,
    5,
    104,
    0,
    0,
    951,
    952,
    5,
    111,
    0,
    0,
    952,
    953,
    5,
    117,
    0,
    0,
    953,
    954,
    5,
    114,
    0,
    0,
    954,
    970,
    5,
    115,
    0,
    0,
    955,
    956,
    5,
    100,
    0,
    0,
    956,
    957,
    5,
    97,
    0,
    0,
    957,
    958,
    5,
    121,
    0,
    0,
    958,
    970,
    5,
    115,
    0,
    0,
    959,
    960,
    5,
    119,
    0,
    0,
    960,
    961,
    5,
    101,
    0,
    0,
    961,
    962,
    5,
    101,
    0,
    0,
    962,
    963,
    5,
    107,
    0,
    0,
    963,
    970,
    5,
    115,
    0,
    0,
    964,
    965,
    5,
    121,
    0,
    0,
    965,
    966,
    5,
    101,
    0,
    0,
    966,
    967,
    5,
    97,
    0,
    0,
    967,
    968,
    5,
    114,
    0,
    0,
    968,
    970,
    5,
    115,
    0,
    0,
    969,
    913,
    1,
    0,
    0,
    0,
    969,
    916,
    1,
    0,
    0,
    0,
    969,
    920,
    1,
    0,
    0,
    0,
    969,
    925,
    1,
    0,
    0,
    0,
    969,
    931,
    1,
    0,
    0,
    0,
    969,
    936,
    1,
    0,
    0,
    0,
    969,
    943,
    1,
    0,
    0,
    0,
    969,
    950,
    1,
    0,
    0,
    0,
    969,
    955,
    1,
    0,
    0,
    0,
    969,
    959,
    1,
    0,
    0,
    0,
    969,
    964,
    1,
    0,
    0,
    0,
    970,
    218,
    1,
    0,
    0,
    0,
    971,
    972,
    5,
    104,
    0,
    0,
    972,
    973,
    5,
    101,
    0,
    0,
    973,
    974,
    5,
    120,
    0,
    0,
    974,
    985,
    1,
    0,
    0,
    0,
    975,
    977,
    5,
    34,
    0,
    0,
    976,
    978,
    3,
    215,
    107,
    0,
    977,
    976,
    1,
    0,
    0,
    0,
    977,
    978,
    1,
    0,
    0,
    0,
    978,
    979,
    1,
    0,
    0,
    0,
    979,
    986,
    5,
    34,
    0,
    0,
    980,
    982,
    5,
    39,
    0,
    0,
    981,
    983,
    3,
    215,
    107,
    0,
    982,
    981,
    1,
    0,
    0,
    0,
    982,
    983,
    1,
    0,
    0,
    0,
    983,
    984,
    1,
    0,
    0,
    0,
    984,
    986,
    5,
    39,
    0,
    0,
    985,
    975,
    1,
    0,
    0,
    0,
    985,
    980,
    1,
    0,
    0,
    0,
    986,
    220,
    1,
    0,
    0,
    0,
    987,
    988,
    7,
    6,
    0,
    0,
    988,
    222,
    1,
    0,
    0,
    0,
    989,
    990,
    5,
    97,
    0,
    0,
    990,
    991,
    5,
    98,
    0,
    0,
    991,
    992,
    5,
    115,
    0,
    0,
    992,
    993,
    5,
    116,
    0,
    0,
    993,
    994,
    5,
    114,
    0,
    0,
    994,
    995,
    5,
    97,
    0,
    0,
    995,
    996,
    5,
    99,
    0,
    0,
    996,
    1078,
    5,
    116,
    0,
    0,
    997,
    998,
    5,
    97,
    0,
    0,
    998,
    999,
    5,
    102,
    0,
    0,
    999,
    1e3,
    5,
    116,
    0,
    0,
    1e3,
    1001,
    5,
    101,
    0,
    0,
    1001,
    1078,
    5,
    114,
    0,
    0,
    1002,
    1003,
    5,
    99,
    0,
    0,
    1003,
    1004,
    5,
    97,
    0,
    0,
    1004,
    1005,
    5,
    115,
    0,
    0,
    1005,
    1078,
    5,
    101,
    0,
    0,
    1006,
    1007,
    5,
    99,
    0,
    0,
    1007,
    1008,
    5,
    97,
    0,
    0,
    1008,
    1009,
    5,
    116,
    0,
    0,
    1009,
    1010,
    5,
    99,
    0,
    0,
    1010,
    1078,
    5,
    104,
    0,
    0,
    1011,
    1012,
    5,
    100,
    0,
    0,
    1012,
    1013,
    5,
    101,
    0,
    0,
    1013,
    1014,
    5,
    102,
    0,
    0,
    1014,
    1015,
    5,
    97,
    0,
    0,
    1015,
    1016,
    5,
    117,
    0,
    0,
    1016,
    1017,
    5,
    108,
    0,
    0,
    1017,
    1078,
    5,
    116,
    0,
    0,
    1018,
    1019,
    5,
    102,
    0,
    0,
    1019,
    1020,
    5,
    105,
    0,
    0,
    1020,
    1021,
    5,
    110,
    0,
    0,
    1021,
    1022,
    5,
    97,
    0,
    0,
    1022,
    1078,
    5,
    108,
    0,
    0,
    1023,
    1024,
    5,
    105,
    0,
    0,
    1024,
    1078,
    5,
    110,
    0,
    0,
    1025,
    1026,
    5,
    105,
    0,
    0,
    1026,
    1027,
    5,
    110,
    0,
    0,
    1027,
    1028,
    5,
    108,
    0,
    0,
    1028,
    1029,
    5,
    105,
    0,
    0,
    1029,
    1030,
    5,
    110,
    0,
    0,
    1030,
    1078,
    5,
    101,
    0,
    0,
    1031,
    1032,
    5,
    108,
    0,
    0,
    1032,
    1033,
    5,
    101,
    0,
    0,
    1033,
    1078,
    5,
    116,
    0,
    0,
    1034,
    1035,
    5,
    109,
    0,
    0,
    1035,
    1036,
    5,
    97,
    0,
    0,
    1036,
    1037,
    5,
    116,
    0,
    0,
    1037,
    1038,
    5,
    99,
    0,
    0,
    1038,
    1078,
    5,
    104,
    0,
    0,
    1039,
    1040,
    5,
    110,
    0,
    0,
    1040,
    1041,
    5,
    117,
    0,
    0,
    1041,
    1042,
    5,
    108,
    0,
    0,
    1042,
    1078,
    5,
    108,
    0,
    0,
    1043,
    1044,
    5,
    111,
    0,
    0,
    1044,
    1078,
    5,
    102,
    0,
    0,
    1045,
    1046,
    5,
    114,
    0,
    0,
    1046,
    1047,
    5,
    101,
    0,
    0,
    1047,
    1048,
    5,
    108,
    0,
    0,
    1048,
    1049,
    5,
    111,
    0,
    0,
    1049,
    1050,
    5,
    99,
    0,
    0,
    1050,
    1051,
    5,
    97,
    0,
    0,
    1051,
    1052,
    5,
    116,
    0,
    0,
    1052,
    1053,
    5,
    97,
    0,
    0,
    1053,
    1054,
    5,
    98,
    0,
    0,
    1054,
    1055,
    5,
    108,
    0,
    0,
    1055,
    1078,
    5,
    101,
    0,
    0,
    1056,
    1057,
    5,
    115,
    0,
    0,
    1057,
    1058,
    5,
    116,
    0,
    0,
    1058,
    1059,
    5,
    97,
    0,
    0,
    1059,
    1060,
    5,
    116,
    0,
    0,
    1060,
    1061,
    5,
    105,
    0,
    0,
    1061,
    1078,
    5,
    99,
    0,
    0,
    1062,
    1063,
    5,
    115,
    0,
    0,
    1063,
    1064,
    5,
    119,
    0,
    0,
    1064,
    1065,
    5,
    105,
    0,
    0,
    1065,
    1066,
    5,
    116,
    0,
    0,
    1066,
    1067,
    5,
    99,
    0,
    0,
    1067,
    1078,
    5,
    104,
    0,
    0,
    1068,
    1069,
    5,
    116,
    0,
    0,
    1069,
    1070,
    5,
    114,
    0,
    0,
    1070,
    1078,
    5,
    121,
    0,
    0,
    1071,
    1072,
    5,
    116,
    0,
    0,
    1072,
    1073,
    5,
    121,
    0,
    0,
    1073,
    1074,
    5,
    112,
    0,
    0,
    1074,
    1075,
    5,
    101,
    0,
    0,
    1075,
    1076,
    5,
    111,
    0,
    0,
    1076,
    1078,
    5,
    102,
    0,
    0,
    1077,
    989,
    1,
    0,
    0,
    0,
    1077,
    997,
    1,
    0,
    0,
    0,
    1077,
    1002,
    1,
    0,
    0,
    0,
    1077,
    1006,
    1,
    0,
    0,
    0,
    1077,
    1011,
    1,
    0,
    0,
    0,
    1077,
    1018,
    1,
    0,
    0,
    0,
    1077,
    1023,
    1,
    0,
    0,
    0,
    1077,
    1025,
    1,
    0,
    0,
    0,
    1077,
    1031,
    1,
    0,
    0,
    0,
    1077,
    1034,
    1,
    0,
    0,
    0,
    1077,
    1039,
    1,
    0,
    0,
    0,
    1077,
    1043,
    1,
    0,
    0,
    0,
    1077,
    1045,
    1,
    0,
    0,
    0,
    1077,
    1056,
    1,
    0,
    0,
    0,
    1077,
    1062,
    1,
    0,
    0,
    0,
    1077,
    1068,
    1,
    0,
    0,
    0,
    1077,
    1071,
    1,
    0,
    0,
    0,
    1078,
    224,
    1,
    0,
    0,
    0,
    1079,
    1080,
    5,
    97,
    0,
    0,
    1080,
    1081,
    5,
    110,
    0,
    0,
    1081,
    1082,
    5,
    111,
    0,
    0,
    1082,
    1083,
    5,
    110,
    0,
    0,
    1083,
    1084,
    5,
    121,
    0,
    0,
    1084,
    1085,
    5,
    109,
    0,
    0,
    1085,
    1086,
    5,
    111,
    0,
    0,
    1086,
    1087,
    5,
    117,
    0,
    0,
    1087,
    1088,
    5,
    115,
    0,
    0,
    1088,
    226,
    1,
    0,
    0,
    0,
    1089,
    1090,
    5,
    98,
    0,
    0,
    1090,
    1091,
    5,
    114,
    0,
    0,
    1091,
    1092,
    5,
    101,
    0,
    0,
    1092,
    1093,
    5,
    97,
    0,
    0,
    1093,
    1094,
    5,
    107,
    0,
    0,
    1094,
    228,
    1,
    0,
    0,
    0,
    1095,
    1096,
    5,
    99,
    0,
    0,
    1096,
    1097,
    5,
    111,
    0,
    0,
    1097,
    1098,
    5,
    110,
    0,
    0,
    1098,
    1099,
    5,
    115,
    0,
    0,
    1099,
    1100,
    5,
    116,
    0,
    0,
    1100,
    1101,
    5,
    97,
    0,
    0,
    1101,
    1102,
    5,
    110,
    0,
    0,
    1102,
    1103,
    5,
    116,
    0,
    0,
    1103,
    230,
    1,
    0,
    0,
    0,
    1104,
    1105,
    5,
    105,
    0,
    0,
    1105,
    1106,
    5,
    109,
    0,
    0,
    1106,
    1107,
    5,
    109,
    0,
    0,
    1107,
    1108,
    5,
    117,
    0,
    0,
    1108,
    1109,
    5,
    116,
    0,
    0,
    1109,
    1110,
    5,
    97,
    0,
    0,
    1110,
    1111,
    5,
    98,
    0,
    0,
    1111,
    1112,
    5,
    108,
    0,
    0,
    1112,
    1113,
    5,
    101,
    0,
    0,
    1113,
    232,
    1,
    0,
    0,
    0,
    1114,
    1115,
    5,
    99,
    0,
    0,
    1115,
    1116,
    5,
    111,
    0,
    0,
    1116,
    1117,
    5,
    110,
    0,
    0,
    1117,
    1118,
    5,
    116,
    0,
    0,
    1118,
    1119,
    5,
    105,
    0,
    0,
    1119,
    1120,
    5,
    110,
    0,
    0,
    1120,
    1121,
    5,
    117,
    0,
    0,
    1121,
    1122,
    5,
    101,
    0,
    0,
    1122,
    234,
    1,
    0,
    0,
    0,
    1123,
    1124,
    5,
    108,
    0,
    0,
    1124,
    1125,
    5,
    101,
    0,
    0,
    1125,
    1126,
    5,
    97,
    0,
    0,
    1126,
    1127,
    5,
    118,
    0,
    0,
    1127,
    1128,
    5,
    101,
    0,
    0,
    1128,
    236,
    1,
    0,
    0,
    0,
    1129,
    1130,
    5,
    101,
    0,
    0,
    1130,
    1131,
    5,
    120,
    0,
    0,
    1131,
    1132,
    5,
    116,
    0,
    0,
    1132,
    1133,
    5,
    101,
    0,
    0,
    1133,
    1134,
    5,
    114,
    0,
    0,
    1134,
    1135,
    5,
    110,
    0,
    0,
    1135,
    1136,
    5,
    97,
    0,
    0,
    1136,
    1137,
    5,
    108,
    0,
    0,
    1137,
    238,
    1,
    0,
    0,
    0,
    1138,
    1139,
    5,
    105,
    0,
    0,
    1139,
    1140,
    5,
    110,
    0,
    0,
    1140,
    1141,
    5,
    100,
    0,
    0,
    1141,
    1142,
    5,
    101,
    0,
    0,
    1142,
    1143,
    5,
    120,
    0,
    0,
    1143,
    1144,
    5,
    101,
    0,
    0,
    1144,
    1145,
    5,
    100,
    0,
    0,
    1145,
    240,
    1,
    0,
    0,
    0,
    1146,
    1147,
    5,
    105,
    0,
    0,
    1147,
    1148,
    5,
    110,
    0,
    0,
    1148,
    1149,
    5,
    116,
    0,
    0,
    1149,
    1150,
    5,
    101,
    0,
    0,
    1150,
    1151,
    5,
    114,
    0,
    0,
    1151,
    1152,
    5,
    110,
    0,
    0,
    1152,
    1153,
    5,
    97,
    0,
    0,
    1153,
    1154,
    5,
    108,
    0,
    0,
    1154,
    242,
    1,
    0,
    0,
    0,
    1155,
    1156,
    5,
    112,
    0,
    0,
    1156,
    1157,
    5,
    97,
    0,
    0,
    1157,
    1158,
    5,
    121,
    0,
    0,
    1158,
    1159,
    5,
    97,
    0,
    0,
    1159,
    1160,
    5,
    98,
    0,
    0,
    1160,
    1161,
    5,
    108,
    0,
    0,
    1161,
    1162,
    5,
    101,
    0,
    0,
    1162,
    244,
    1,
    0,
    0,
    0,
    1163,
    1164,
    5,
    112,
    0,
    0,
    1164,
    1165,
    5,
    114,
    0,
    0,
    1165,
    1166,
    5,
    105,
    0,
    0,
    1166,
    1167,
    5,
    118,
    0,
    0,
    1167,
    1168,
    5,
    97,
    0,
    0,
    1168,
    1169,
    5,
    116,
    0,
    0,
    1169,
    1170,
    5,
    101,
    0,
    0,
    1170,
    246,
    1,
    0,
    0,
    0,
    1171,
    1172,
    5,
    112,
    0,
    0,
    1172,
    1173,
    5,
    117,
    0,
    0,
    1173,
    1174,
    5,
    98,
    0,
    0,
    1174,
    1175,
    5,
    108,
    0,
    0,
    1175,
    1176,
    5,
    105,
    0,
    0,
    1176,
    1177,
    5,
    99,
    0,
    0,
    1177,
    248,
    1,
    0,
    0,
    0,
    1178,
    1179,
    5,
    118,
    0,
    0,
    1179,
    1180,
    5,
    105,
    0,
    0,
    1180,
    1181,
    5,
    114,
    0,
    0,
    1181,
    1182,
    5,
    116,
    0,
    0,
    1182,
    1183,
    5,
    117,
    0,
    0,
    1183,
    1184,
    5,
    97,
    0,
    0,
    1184,
    1185,
    5,
    108,
    0,
    0,
    1185,
    250,
    1,
    0,
    0,
    0,
    1186,
    1187,
    5,
    112,
    0,
    0,
    1187,
    1188,
    5,
    117,
    0,
    0,
    1188,
    1189,
    5,
    114,
    0,
    0,
    1189,
    1190,
    5,
    101,
    0,
    0,
    1190,
    252,
    1,
    0,
    0,
    0,
    1191,
    1192,
    5,
    116,
    0,
    0,
    1192,
    1193,
    5,
    121,
    0,
    0,
    1193,
    1194,
    5,
    112,
    0,
    0,
    1194,
    1195,
    5,
    101,
    0,
    0,
    1195,
    254,
    1,
    0,
    0,
    0,
    1196,
    1197,
    5,
    118,
    0,
    0,
    1197,
    1198,
    5,
    105,
    0,
    0,
    1198,
    1199,
    5,
    101,
    0,
    0,
    1199,
    1200,
    5,
    119,
    0,
    0,
    1200,
    256,
    1,
    0,
    0,
    0,
    1201,
    1202,
    5,
    103,
    0,
    0,
    1202,
    1203,
    5,
    108,
    0,
    0,
    1203,
    1204,
    5,
    111,
    0,
    0,
    1204,
    1205,
    5,
    98,
    0,
    0,
    1205,
    1206,
    5,
    97,
    0,
    0,
    1206,
    1207,
    5,
    108,
    0,
    0,
    1207,
    258,
    1,
    0,
    0,
    0,
    1208,
    1209,
    5,
    99,
    0,
    0,
    1209,
    1210,
    5,
    111,
    0,
    0,
    1210,
    1211,
    5,
    110,
    0,
    0,
    1211,
    1212,
    5,
    115,
    0,
    0,
    1212,
    1213,
    5,
    116,
    0,
    0,
    1213,
    1214,
    5,
    114,
    0,
    0,
    1214,
    1215,
    5,
    117,
    0,
    0,
    1215,
    1216,
    5,
    99,
    0,
    0,
    1216,
    1217,
    5,
    116,
    0,
    0,
    1217,
    1218,
    5,
    111,
    0,
    0,
    1218,
    1219,
    5,
    114,
    0,
    0,
    1219,
    260,
    1,
    0,
    0,
    0,
    1220,
    1221,
    5,
    102,
    0,
    0,
    1221,
    1222,
    5,
    97,
    0,
    0,
    1222,
    1223,
    5,
    108,
    0,
    0,
    1223,
    1224,
    5,
    108,
    0,
    0,
    1224,
    1225,
    5,
    98,
    0,
    0,
    1225,
    1226,
    5,
    97,
    0,
    0,
    1226,
    1227,
    5,
    99,
    0,
    0,
    1227,
    1228,
    5,
    107,
    0,
    0,
    1228,
    262,
    1,
    0,
    0,
    0,
    1229,
    1230,
    5,
    114,
    0,
    0,
    1230,
    1231,
    5,
    101,
    0,
    0,
    1231,
    1232,
    5,
    99,
    0,
    0,
    1232,
    1233,
    5,
    101,
    0,
    0,
    1233,
    1234,
    5,
    105,
    0,
    0,
    1234,
    1235,
    5,
    118,
    0,
    0,
    1235,
    1236,
    5,
    101,
    0,
    0,
    1236,
    264,
    1,
    0,
    0,
    0,
    1237,
    1241,
    3,
    267,
    133,
    0,
    1238,
    1240,
    3,
    269,
    134,
    0,
    1239,
    1238,
    1,
    0,
    0,
    0,
    1240,
    1243,
    1,
    0,
    0,
    0,
    1241,
    1239,
    1,
    0,
    0,
    0,
    1241,
    1242,
    1,
    0,
    0,
    0,
    1242,
    266,
    1,
    0,
    0,
    0,
    1243,
    1241,
    1,
    0,
    0,
    0,
    1244,
    1245,
    7,
    7,
    0,
    0,
    1245,
    268,
    1,
    0,
    0,
    0,
    1246,
    1247,
    7,
    8,
    0,
    0,
    1247,
    270,
    1,
    0,
    0,
    0,
    1248,
    1249,
    5,
    117,
    0,
    0,
    1249,
    1250,
    5,
    110,
    0,
    0,
    1250,
    1251,
    5,
    105,
    0,
    0,
    1251,
    1252,
    5,
    99,
    0,
    0,
    1252,
    1253,
    5,
    111,
    0,
    0,
    1253,
    1254,
    5,
    100,
    0,
    0,
    1254,
    1256,
    5,
    101,
    0,
    0,
    1255,
    1248,
    1,
    0,
    0,
    0,
    1255,
    1256,
    1,
    0,
    0,
    0,
    1256,
    1273,
    1,
    0,
    0,
    0,
    1257,
    1261,
    5,
    34,
    0,
    0,
    1258,
    1260,
    3,
    273,
    136,
    0,
    1259,
    1258,
    1,
    0,
    0,
    0,
    1260,
    1263,
    1,
    0,
    0,
    0,
    1261,
    1259,
    1,
    0,
    0,
    0,
    1261,
    1262,
    1,
    0,
    0,
    0,
    1262,
    1264,
    1,
    0,
    0,
    0,
    1263,
    1261,
    1,
    0,
    0,
    0,
    1264,
    1274,
    5,
    34,
    0,
    0,
    1265,
    1269,
    5,
    39,
    0,
    0,
    1266,
    1268,
    3,
    275,
    137,
    0,
    1267,
    1266,
    1,
    0,
    0,
    0,
    1268,
    1271,
    1,
    0,
    0,
    0,
    1269,
    1267,
    1,
    0,
    0,
    0,
    1269,
    1270,
    1,
    0,
    0,
    0,
    1270,
    1272,
    1,
    0,
    0,
    0,
    1271,
    1269,
    1,
    0,
    0,
    0,
    1272,
    1274,
    5,
    39,
    0,
    0,
    1273,
    1257,
    1,
    0,
    0,
    0,
    1273,
    1265,
    1,
    0,
    0,
    0,
    1274,
    272,
    1,
    0,
    0,
    0,
    1275,
    1279,
    8,
    9,
    0,
    0,
    1276,
    1277,
    5,
    92,
    0,
    0,
    1277,
    1279,
    9,
    0,
    0,
    0,
    1278,
    1275,
    1,
    0,
    0,
    0,
    1278,
    1276,
    1,
    0,
    0,
    0,
    1279,
    274,
    1,
    0,
    0,
    0,
    1280,
    1284,
    8,
    10,
    0,
    0,
    1281,
    1282,
    5,
    92,
    0,
    0,
    1282,
    1284,
    9,
    0,
    0,
    0,
    1283,
    1280,
    1,
    0,
    0,
    0,
    1283,
    1281,
    1,
    0,
    0,
    0,
    1284,
    276,
    1,
    0,
    0,
    0,
    1285,
    1287,
    7,
    0,
    0,
    0,
    1286,
    1285,
    1,
    0,
    0,
    0,
    1287,
    1288,
    1,
    0,
    0,
    0,
    1288,
    1286,
    1,
    0,
    0,
    0,
    1288,
    1289,
    1,
    0,
    0,
    0,
    1289,
    1290,
    1,
    0,
    0,
    0,
    1290,
    1292,
    5,
    46,
    0,
    0,
    1291,
    1293,
    7,
    0,
    0,
    0,
    1292,
    1291,
    1,
    0,
    0,
    0,
    1293,
    1294,
    1,
    0,
    0,
    0,
    1294,
    1292,
    1,
    0,
    0,
    0,
    1294,
    1295,
    1,
    0,
    0,
    0,
    1295,
    1302,
    1,
    0,
    0,
    0,
    1296,
    1298,
    5,
    46,
    0,
    0,
    1297,
    1299,
    7,
    0,
    0,
    0,
    1298,
    1297,
    1,
    0,
    0,
    0,
    1299,
    1300,
    1,
    0,
    0,
    0,
    1300,
    1298,
    1,
    0,
    0,
    0,
    1300,
    1301,
    1,
    0,
    0,
    0,
    1301,
    1303,
    1,
    0,
    0,
    0,
    1302,
    1296,
    1,
    0,
    0,
    0,
    1302,
    1303,
    1,
    0,
    0,
    0,
    1303,
    278,
    1,
    0,
    0,
    0,
    1304,
    1306,
    7,
    11,
    0,
    0,
    1305,
    1304,
    1,
    0,
    0,
    0,
    1306,
    1307,
    1,
    0,
    0,
    0,
    1307,
    1305,
    1,
    0,
    0,
    0,
    1307,
    1308,
    1,
    0,
    0,
    0,
    1308,
    1309,
    1,
    0,
    0,
    0,
    1309,
    1310,
    6,
    139,
    0,
    0,
    1310,
    280,
    1,
    0,
    0,
    0,
    1311,
    1312,
    5,
    47,
    0,
    0,
    1312,
    1313,
    5,
    42,
    0,
    0,
    1313,
    1317,
    1,
    0,
    0,
    0,
    1314,
    1316,
    9,
    0,
    0,
    0,
    1315,
    1314,
    1,
    0,
    0,
    0,
    1316,
    1319,
    1,
    0,
    0,
    0,
    1317,
    1318,
    1,
    0,
    0,
    0,
    1317,
    1315,
    1,
    0,
    0,
    0,
    1318,
    1320,
    1,
    0,
    0,
    0,
    1319,
    1317,
    1,
    0,
    0,
    0,
    1320,
    1321,
    5,
    42,
    0,
    0,
    1321,
    1322,
    5,
    47,
    0,
    0,
    1322,
    1323,
    1,
    0,
    0,
    0,
    1323,
    1324,
    6,
    140,
    1,
    0,
    1324,
    282,
    1,
    0,
    0,
    0,
    1325,
    1326,
    5,
    47,
    0,
    0,
    1326,
    1327,
    5,
    47,
    0,
    0,
    1327,
    1331,
    1,
    0,
    0,
    0,
    1328,
    1330,
    8,
    12,
    0,
    0,
    1329,
    1328,
    1,
    0,
    0,
    0,
    1330,
    1333,
    1,
    0,
    0,
    0,
    1331,
    1329,
    1,
    0,
    0,
    0,
    1331,
    1332,
    1,
    0,
    0,
    0,
    1332,
    1334,
    1,
    0,
    0,
    0,
    1333,
    1331,
    1,
    0,
    0,
    0,
    1334,
    1335,
    6,
    141,
    1,
    0,
    1335,
    284,
    1,
    0,
    0,
    0,
    38,
    0,
    721,
    729,
    738,
    751,
    753,
    767,
    769,
    854,
    861,
    872,
    876,
    880,
    884,
    887,
    891,
    896,
    905,
    910,
    969,
    977,
    982,
    985,
    1077,
    1241,
    1255,
    1261,
    1269,
    1273,
    1278,
    1283,
    1288,
    1294,
    1300,
    1302,
    1307,
    1317,
    1331,
    2,
    6,
    0,
    0,
    0,
    1,
    0
  ];
  SolidityLexer.DecisionsToDFA = _SolidityLexer._ATN.decisionToState.map((ds, index) => new u(ds, index));
  var SolidityLexer_default = SolidityLexer;

  // src/antlr/SolidityParser.ts
  var _SolidityParser = class extends I {
    get grammarFileName() {
      return "Solidity.g4";
    }
    get literalNames() {
      return _SolidityParser.literalNames;
    }
    get symbolicNames() {
      return _SolidityParser.symbolicNames;
    }
    get ruleNames() {
      return _SolidityParser.ruleNames;
    }
    get serializedATN() {
      return _SolidityParser._serializedATN;
    }
    createFailedPredicateException(predicate, message) {
      return new f(this, predicate, message);
    }
    constructor(input) {
      super(input);
      this._interp = new k(this, _SolidityParser._ATN, _SolidityParser.DecisionsToDFA, new O());
    }
    sourceUnit() {
      let localctx = new SourceUnitContext(this, this._ctx, this.state);
      this.enterRule(localctx, 0, _SolidityParser.RULE_sourceUnit);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 215;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          while ((_la & ~31) === 0 && (1 << _la & 104620034) !== 0 || (_la - 36 & ~31) === 0 && (1 << _la - 36 & 2080392501) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 3896770685) !== 0 || _la === 127 || _la === 128) {
            {
              this.state = 213;
              this._errHandler.sync(this);
              switch (this._interp.adaptivePredict(this._input, 0, this._ctx)) {
                case 1:
                  {
                    this.state = 202;
                    this.pragmaDirective();
                  }
                  break;
                case 2:
                  {
                    this.state = 203;
                    this.importDirective();
                  }
                  break;
                case 3:
                  {
                    this.state = 204;
                    this.contractDefinition();
                  }
                  break;
                case 4:
                  {
                    this.state = 205;
                    this.enumDefinition();
                  }
                  break;
                case 5:
                  {
                    this.state = 206;
                    this.eventDefinition();
                  }
                  break;
                case 6:
                  {
                    this.state = 207;
                    this.structDefinition();
                  }
                  break;
                case 7:
                  {
                    this.state = 208;
                    this.functionDefinition();
                  }
                  break;
                case 8:
                  {
                    this.state = 209;
                    this.fileLevelConstant();
                  }
                  break;
                case 9:
                  {
                    this.state = 210;
                    this.customErrorDefinition();
                  }
                  break;
                case 10:
                  {
                    this.state = 211;
                    this.typeDefinition();
                  }
                  break;
                case 11:
                  {
                    this.state = 212;
                    this.usingForDeclaration();
                  }
                  break;
              }
            }
            this.state = 217;
            this._errHandler.sync(this);
            _la = this._input.LA(1);
          }
          this.state = 218;
          this.match(_SolidityParser.EOF);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    pragmaDirective() {
      let localctx = new PragmaDirectiveContext(this, this._ctx, this.state);
      this.enterRule(localctx, 2, _SolidityParser.RULE_pragmaDirective);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 220;
          this.match(_SolidityParser.T__0);
          this.state = 221;
          this.pragmaName();
          this.state = 222;
          this.pragmaValue();
          this.state = 223;
          this.match(_SolidityParser.T__1);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    pragmaName() {
      let localctx = new PragmaNameContext(this, this._ctx, this.state);
      this.enterRule(localctx, 4, _SolidityParser.RULE_pragmaName);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 225;
          this.identifier();
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    pragmaValue() {
      let localctx = new PragmaValueContext(this, this._ctx, this.state);
      this.enterRule(localctx, 6, _SolidityParser.RULE_pragmaValue);
      try {
        this.state = 230;
        this._errHandler.sync(this);
        switch (this._interp.adaptivePredict(this._input, 2, this._ctx)) {
          case 1:
            this.enterOuterAlt(localctx, 1);
            {
              this.state = 227;
              this.match(_SolidityParser.T__2);
            }
            break;
          case 2:
            this.enterOuterAlt(localctx, 2);
            {
              this.state = 228;
              this.version();
            }
            break;
          case 3:
            this.enterOuterAlt(localctx, 3);
            {
              this.state = 229;
              this.expression(0);
            }
            break;
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    version() {
      let localctx = new VersionContext(this, this._ctx, this.state);
      this.enterRule(localctx, 8, _SolidityParser.RULE_version);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 232;
          this.versionConstraint();
          this.state = 239;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          while ((_la & ~31) === 0 && (1 << _la & 4080) !== 0 || _la === 103 || _la === 130) {
            {
              {
                this.state = 234;
                this._errHandler.sync(this);
                _la = this._input.LA(1);
                if (_la === 4) {
                  {
                    this.state = 233;
                    this.match(_SolidityParser.T__3);
                  }
                }
                this.state = 236;
                this.versionConstraint();
              }
            }
            this.state = 241;
            this._errHandler.sync(this);
            _la = this._input.LA(1);
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    versionOperator() {
      let localctx = new VersionOperatorContext(this, this._ctx, this.state);
      this.enterRule(localctx, 10, _SolidityParser.RULE_versionOperator);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 242;
          _la = this._input.LA(1);
          if (!((_la & ~31) === 0 && (1 << _la & 4064) !== 0)) {
            this._errHandler.recoverInline(this);
          } else {
            this._errHandler.reportMatch(this);
            this.consume();
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    versionConstraint() {
      let localctx = new VersionConstraintContext(this, this._ctx, this.state);
      this.enterRule(localctx, 12, _SolidityParser.RULE_versionConstraint);
      let _la;
      try {
        this.state = 252;
        this._errHandler.sync(this);
        switch (this._interp.adaptivePredict(this._input, 7, this._ctx)) {
          case 1:
            this.enterOuterAlt(localctx, 1);
            {
              this.state = 245;
              this._errHandler.sync(this);
              _la = this._input.LA(1);
              if ((_la & ~31) === 0 && (1 << _la & 4064) !== 0) {
                {
                  this.state = 244;
                  this.versionOperator();
                }
              }
              this.state = 247;
              this.match(_SolidityParser.VersionLiteral);
            }
            break;
          case 2:
            this.enterOuterAlt(localctx, 2);
            {
              this.state = 249;
              this._errHandler.sync(this);
              _la = this._input.LA(1);
              if ((_la & ~31) === 0 && (1 << _la & 4064) !== 0) {
                {
                  this.state = 248;
                  this.versionOperator();
                }
              }
              this.state = 251;
              this.match(_SolidityParser.DecimalNumber);
            }
            break;
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    importDeclaration() {
      let localctx = new ImportDeclarationContext(this, this._ctx, this.state);
      this.enterRule(localctx, 14, _SolidityParser.RULE_importDeclaration);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 254;
          this.identifier();
          this.state = 257;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 12) {
            {
              this.state = 255;
              this.match(_SolidityParser.T__11);
              this.state = 256;
              this.identifier();
            }
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    importDirective() {
      let localctx = new ImportDirectiveContext(this, this._ctx, this.state);
      this.enterRule(localctx, 16, _SolidityParser.RULE_importDirective);
      let _la;
      try {
        this.state = 295;
        this._errHandler.sync(this);
        switch (this._interp.adaptivePredict(this._input, 13, this._ctx)) {
          case 1:
            this.enterOuterAlt(localctx, 1);
            {
              this.state = 259;
              this.match(_SolidityParser.T__12);
              this.state = 260;
              this.importPath();
              this.state = 263;
              this._errHandler.sync(this);
              _la = this._input.LA(1);
              if (_la === 12) {
                {
                  this.state = 261;
                  this.match(_SolidityParser.T__11);
                  this.state = 262;
                  this.identifier();
                }
              }
              this.state = 265;
              this.match(_SolidityParser.T__1);
            }
            break;
          case 2:
            this.enterOuterAlt(localctx, 2);
            {
              this.state = 267;
              this.match(_SolidityParser.T__12);
              this.state = 270;
              this._errHandler.sync(this);
              switch (this._input.LA(1)) {
                case 3:
                  {
                    this.state = 268;
                    this.match(_SolidityParser.T__2);
                  }
                  break;
                case 14:
                case 25:
                case 44:
                case 50:
                case 62:
                case 95:
                case 113:
                case 117:
                case 124:
                case 125:
                case 127:
                case 128:
                  {
                    this.state = 269;
                    this.identifier();
                  }
                  break;
                default:
                  throw new A(this);
              }
              this.state = 274;
              this._errHandler.sync(this);
              _la = this._input.LA(1);
              if (_la === 12) {
                {
                  this.state = 272;
                  this.match(_SolidityParser.T__11);
                  this.state = 273;
                  this.identifier();
                }
              }
              this.state = 276;
              this.match(_SolidityParser.T__13);
              this.state = 277;
              this.importPath();
              this.state = 278;
              this.match(_SolidityParser.T__1);
            }
            break;
          case 3:
            this.enterOuterAlt(localctx, 3);
            {
              this.state = 280;
              this.match(_SolidityParser.T__12);
              this.state = 281;
              this.match(_SolidityParser.T__14);
              this.state = 282;
              this.importDeclaration();
              this.state = 287;
              this._errHandler.sync(this);
              _la = this._input.LA(1);
              while (_la === 16) {
                {
                  {
                    this.state = 283;
                    this.match(_SolidityParser.T__15);
                    this.state = 284;
                    this.importDeclaration();
                  }
                }
                this.state = 289;
                this._errHandler.sync(this);
                _la = this._input.LA(1);
              }
              this.state = 290;
              this.match(_SolidityParser.T__16);
              this.state = 291;
              this.match(_SolidityParser.T__13);
              this.state = 292;
              this.importPath();
              this.state = 293;
              this.match(_SolidityParser.T__1);
            }
            break;
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    importPath() {
      let localctx = new ImportPathContext(this, this._ctx, this.state);
      this.enterRule(localctx, 18, _SolidityParser.RULE_importPath);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 297;
          this.match(_SolidityParser.StringLiteralFragment);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    contractDefinition() {
      let localctx = new ContractDefinitionContext(this, this._ctx, this.state);
      this.enterRule(localctx, 20, _SolidityParser.RULE_contractDefinition);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 300;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 18) {
            {
              this.state = 299;
              this.match(_SolidityParser.T__17);
            }
          }
          this.state = 302;
          _la = this._input.LA(1);
          if (!((_la & ~31) === 0 && (1 << _la & 3670016) !== 0)) {
            this._errHandler.recoverInline(this);
          } else {
            this._errHandler.reportMatch(this);
            this.consume();
          }
          this.state = 303;
          this.identifier();
          this.state = 313;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 22) {
            {
              this.state = 304;
              this.match(_SolidityParser.T__21);
              this.state = 305;
              this.inheritanceSpecifier();
              this.state = 310;
              this._errHandler.sync(this);
              _la = this._input.LA(1);
              while (_la === 16) {
                {
                  {
                    this.state = 306;
                    this.match(_SolidityParser.T__15);
                    this.state = 307;
                    this.inheritanceSpecifier();
                  }
                }
                this.state = 312;
                this._errHandler.sync(this);
                _la = this._input.LA(1);
              }
            }
          }
          this.state = 315;
          this.match(_SolidityParser.T__14);
          this.state = 319;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          while ((_la & ~31) === 0 && (1 << _la & 100679680) !== 0 || (_la - 36 & ~31) === 0 && (1 << _la - 36 & 2080392503) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 3896770685) !== 0 || _la === 127 || _la === 128) {
            {
              {
                this.state = 316;
                this.contractPart();
              }
            }
            this.state = 321;
            this._errHandler.sync(this);
            _la = this._input.LA(1);
          }
          this.state = 322;
          this.match(_SolidityParser.T__16);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    inheritanceSpecifier() {
      let localctx = new InheritanceSpecifierContext(this, this._ctx, this.state);
      this.enterRule(localctx, 22, _SolidityParser.RULE_inheritanceSpecifier);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 324;
          this.userDefinedTypeName();
          this.state = 330;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 23) {
            {
              this.state = 325;
              this.match(_SolidityParser.T__22);
              this.state = 327;
              this._errHandler.sync(this);
              _la = this._input.LA(1);
              if ((_la & ~31) === 0 && (1 << _la & 3263184960) !== 0 || (_la - 38 & ~31) === 0 && (1 << _la - 38 & 4278194513) !== 0 || (_la - 71 & ~31) === 0 && (1 << _la - 71 & 4244635651) !== 0 || (_la - 103 & ~31) === 0 && (1 << _la - 103 & 124273675) !== 0) {
                {
                  this.state = 326;
                  this.expressionList();
                }
              }
              this.state = 329;
              this.match(_SolidityParser.T__23);
            }
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    contractPart() {
      let localctx = new ContractPartContext(this, this._ctx, this.state);
      this.enterRule(localctx, 24, _SolidityParser.RULE_contractPart);
      try {
        this.state = 341;
        this._errHandler.sync(this);
        switch (this._interp.adaptivePredict(this._input, 20, this._ctx)) {
          case 1:
            this.enterOuterAlt(localctx, 1);
            {
              this.state = 332;
              this.stateVariableDeclaration();
            }
            break;
          case 2:
            this.enterOuterAlt(localctx, 2);
            {
              this.state = 333;
              this.usingForDeclaration();
            }
            break;
          case 3:
            this.enterOuterAlt(localctx, 3);
            {
              this.state = 334;
              this.structDefinition();
            }
            break;
          case 4:
            this.enterOuterAlt(localctx, 4);
            {
              this.state = 335;
              this.modifierDefinition();
            }
            break;
          case 5:
            this.enterOuterAlt(localctx, 5);
            {
              this.state = 336;
              this.functionDefinition();
            }
            break;
          case 6:
            this.enterOuterAlt(localctx, 6);
            {
              this.state = 337;
              this.eventDefinition();
            }
            break;
          case 7:
            this.enterOuterAlt(localctx, 7);
            {
              this.state = 338;
              this.enumDefinition();
            }
            break;
          case 8:
            this.enterOuterAlt(localctx, 8);
            {
              this.state = 339;
              this.customErrorDefinition();
            }
            break;
          case 9:
            this.enterOuterAlt(localctx, 9);
            {
              this.state = 340;
              this.typeDefinition();
            }
            break;
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    stateVariableDeclaration() {
      let localctx = new StateVariableDeclarationContext(this, this._ctx, this.state);
      this.enterRule(localctx, 26, _SolidityParser.RULE_stateVariableDeclaration);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 343;
          this.typeName(0);
          this.state = 352;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          while ((_la - 96 & ~31) === 0 && (1 << _la - 96 & 13680641) !== 0) {
            {
              this.state = 350;
              this._errHandler.sync(this);
              switch (this._input.LA(1)) {
                case 119:
                  {
                    this.state = 344;
                    this.match(_SolidityParser.PublicKeyword);
                  }
                  break;
                case 116:
                  {
                    this.state = 345;
                    this.match(_SolidityParser.InternalKeyword);
                  }
                  break;
                case 118:
                  {
                    this.state = 346;
                    this.match(_SolidityParser.PrivateKeyword);
                  }
                  break;
                case 110:
                  {
                    this.state = 347;
                    this.match(_SolidityParser.ConstantKeyword);
                  }
                  break;
                case 111:
                  {
                    this.state = 348;
                    this.match(_SolidityParser.ImmutableKeyword);
                  }
                  break;
                case 96:
                  {
                    this.state = 349;
                    this.overrideSpecifier();
                  }
                  break;
                default:
                  throw new A(this);
              }
            }
            this.state = 354;
            this._errHandler.sync(this);
            _la = this._input.LA(1);
          }
          this.state = 355;
          this.identifier();
          this.state = 358;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 11) {
            {
              this.state = 356;
              this.match(_SolidityParser.T__10);
              this.state = 357;
              this.expression(0);
            }
          }
          this.state = 360;
          this.match(_SolidityParser.T__1);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    fileLevelConstant() {
      let localctx = new FileLevelConstantContext(this, this._ctx, this.state);
      this.enterRule(localctx, 28, _SolidityParser.RULE_fileLevelConstant);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 362;
          this.typeName(0);
          this.state = 363;
          this.match(_SolidityParser.ConstantKeyword);
          this.state = 364;
          this.identifier();
          this.state = 365;
          this.match(_SolidityParser.T__10);
          this.state = 366;
          this.expression(0);
          this.state = 367;
          this.match(_SolidityParser.T__1);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    customErrorDefinition() {
      let localctx = new CustomErrorDefinitionContext(this, this._ctx, this.state);
      this.enterRule(localctx, 30, _SolidityParser.RULE_customErrorDefinition);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 369;
          this.match(_SolidityParser.T__24);
          this.state = 370;
          this.identifier();
          this.state = 371;
          this.parameterList();
          this.state = 372;
          this.match(_SolidityParser.T__1);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    typeDefinition() {
      let localctx = new TypeDefinitionContext(this, this._ctx, this.state);
      this.enterRule(localctx, 32, _SolidityParser.RULE_typeDefinition);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 374;
          this.match(_SolidityParser.TypeKeyword);
          this.state = 375;
          this.identifier();
          this.state = 376;
          this.match(_SolidityParser.T__21);
          this.state = 377;
          this.elementaryTypeName();
          this.state = 378;
          this.match(_SolidityParser.T__1);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    usingForDeclaration() {
      let localctx = new UsingForDeclarationContext(this, this._ctx, this.state);
      this.enterRule(localctx, 34, _SolidityParser.RULE_usingForDeclaration);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 380;
          this.match(_SolidityParser.T__25);
          this.state = 381;
          this.usingForObject();
          this.state = 382;
          this.match(_SolidityParser.T__26);
          this.state = 385;
          this._errHandler.sync(this);
          switch (this._input.LA(1)) {
            case 3:
              {
                this.state = 383;
                this.match(_SolidityParser.T__2);
              }
              break;
            case 14:
            case 25:
            case 38:
            case 44:
            case 46:
            case 50:
            case 62:
            case 63:
            case 64:
            case 65:
            case 66:
            case 95:
            case 97:
            case 98:
            case 99:
            case 100:
            case 101:
            case 113:
            case 117:
            case 124:
            case 125:
            case 127:
            case 128:
              {
                this.state = 384;
                this.typeName(0);
              }
              break;
            default:
              throw new A(this);
          }
          this.state = 388;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 124) {
            {
              this.state = 387;
              this.match(_SolidityParser.GlobalKeyword);
            }
          }
          this.state = 390;
          this.match(_SolidityParser.T__1);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    usingForObject() {
      let localctx = new UsingForObjectContext(this, this._ctx, this.state);
      this.enterRule(localctx, 36, _SolidityParser.RULE_usingForObject);
      let _la;
      try {
        this.state = 404;
        this._errHandler.sync(this);
        switch (this._input.LA(1)) {
          case 14:
          case 25:
          case 44:
          case 50:
          case 62:
          case 95:
          case 113:
          case 117:
          case 124:
          case 125:
          case 127:
          case 128:
            this.enterOuterAlt(localctx, 1);
            {
              this.state = 392;
              this.userDefinedTypeName();
            }
            break;
          case 15:
            this.enterOuterAlt(localctx, 2);
            {
              this.state = 393;
              this.match(_SolidityParser.T__14);
              this.state = 394;
              this.usingForObjectDirective();
              this.state = 399;
              this._errHandler.sync(this);
              _la = this._input.LA(1);
              while (_la === 16) {
                {
                  {
                    this.state = 395;
                    this.match(_SolidityParser.T__15);
                    this.state = 396;
                    this.usingForObjectDirective();
                  }
                }
                this.state = 401;
                this._errHandler.sync(this);
                _la = this._input.LA(1);
              }
              this.state = 402;
              this.match(_SolidityParser.T__16);
            }
            break;
          default:
            throw new A(this);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    usingForObjectDirective() {
      let localctx = new UsingForObjectDirectiveContext(this, this._ctx, this.state);
      this.enterRule(localctx, 38, _SolidityParser.RULE_usingForObjectDirective);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 406;
          this.userDefinedTypeName();
          this.state = 409;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 12) {
            {
              this.state = 407;
              this.match(_SolidityParser.T__11);
              this.state = 408;
              this.userDefinableOperators();
            }
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    userDefinableOperators() {
      let localctx = new UserDefinableOperatorsContext(this, this._ctx, this.state);
      this.enterRule(localctx, 40, _SolidityParser.RULE_userDefinableOperators);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 411;
          _la = this._input.LA(1);
          if (!((_la & ~31) === 0 && (1 << _la & 4026533864) !== 0 || (_la - 32 & ~31) === 0 && (1 << _la - 32 & 15) !== 0)) {
            this._errHandler.recoverInline(this);
          } else {
            this._errHandler.reportMatch(this);
            this.consume();
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    structDefinition() {
      let localctx = new StructDefinitionContext(this, this._ctx, this.state);
      this.enterRule(localctx, 42, _SolidityParser.RULE_structDefinition);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 413;
          this.match(_SolidityParser.T__35);
          this.state = 414;
          this.identifier();
          this.state = 415;
          this.match(_SolidityParser.T__14);
          this.state = 426;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 14 || _la === 25 || (_la - 38 & ~31) === 0 && (1 << _la - 38 & 520098113) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 1615069309) !== 0 || _la === 127 || _la === 128) {
            {
              this.state = 416;
              this.variableDeclaration();
              this.state = 417;
              this.match(_SolidityParser.T__1);
              this.state = 423;
              this._errHandler.sync(this);
              _la = this._input.LA(1);
              while (_la === 14 || _la === 25 || (_la - 38 & ~31) === 0 && (1 << _la - 38 & 520098113) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 1615069309) !== 0 || _la === 127 || _la === 128) {
                {
                  {
                    this.state = 418;
                    this.variableDeclaration();
                    this.state = 419;
                    this.match(_SolidityParser.T__1);
                  }
                }
                this.state = 425;
                this._errHandler.sync(this);
                _la = this._input.LA(1);
              }
            }
          }
          this.state = 428;
          this.match(_SolidityParser.T__16);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    modifierDefinition() {
      let localctx = new ModifierDefinitionContext(this, this._ctx, this.state);
      this.enterRule(localctx, 44, _SolidityParser.RULE_modifierDefinition);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 430;
          this.match(_SolidityParser.T__36);
          this.state = 431;
          this.identifier();
          this.state = 433;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 23) {
            {
              this.state = 432;
              this.parameterList();
            }
          }
          this.state = 439;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          while (_la === 96 || _la === 120) {
            {
              this.state = 437;
              this._errHandler.sync(this);
              switch (this._input.LA(1)) {
                case 120:
                  {
                    this.state = 435;
                    this.match(_SolidityParser.VirtualKeyword);
                  }
                  break;
                case 96:
                  {
                    this.state = 436;
                    this.overrideSpecifier();
                  }
                  break;
                default:
                  throw new A(this);
              }
            }
            this.state = 441;
            this._errHandler.sync(this);
            _la = this._input.LA(1);
          }
          this.state = 444;
          this._errHandler.sync(this);
          switch (this._input.LA(1)) {
            case 2:
              {
                this.state = 442;
                this.match(_SolidityParser.T__1);
              }
              break;
            case 15:
              {
                this.state = 443;
                this.block();
              }
              break;
            default:
              throw new A(this);
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    modifierInvocation() {
      let localctx = new ModifierInvocationContext(this, this._ctx, this.state);
      this.enterRule(localctx, 46, _SolidityParser.RULE_modifierInvocation);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 446;
          this.identifier();
          this.state = 452;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 23) {
            {
              this.state = 447;
              this.match(_SolidityParser.T__22);
              this.state = 449;
              this._errHandler.sync(this);
              _la = this._input.LA(1);
              if ((_la & ~31) === 0 && (1 << _la & 3263184960) !== 0 || (_la - 38 & ~31) === 0 && (1 << _la - 38 & 4278194513) !== 0 || (_la - 71 & ~31) === 0 && (1 << _la - 71 & 4244635651) !== 0 || (_la - 103 & ~31) === 0 && (1 << _la - 103 & 124273675) !== 0) {
                {
                  this.state = 448;
                  this.expressionList();
                }
              }
              this.state = 451;
              this.match(_SolidityParser.T__23);
            }
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    functionDefinition() {
      let localctx = new FunctionDefinitionContext(this, this._ctx, this.state);
      this.enterRule(localctx, 48, _SolidityParser.RULE_functionDefinition);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 454;
          this.functionDescriptor();
          this.state = 455;
          this.parameterList();
          this.state = 456;
          this.modifierList();
          this.state = 458;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 39) {
            {
              this.state = 457;
              this.returnParameters();
            }
          }
          this.state = 462;
          this._errHandler.sync(this);
          switch (this._input.LA(1)) {
            case 2:
              {
                this.state = 460;
                this.match(_SolidityParser.T__1);
              }
              break;
            case 15:
              {
                this.state = 461;
                this.block();
              }
              break;
            default:
              throw new A(this);
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    functionDescriptor() {
      let localctx = new FunctionDescriptorContext(this, this._ctx, this.state);
      this.enterRule(localctx, 50, _SolidityParser.RULE_functionDescriptor);
      let _la;
      try {
        this.state = 471;
        this._errHandler.sync(this);
        switch (this._input.LA(1)) {
          case 38:
            this.enterOuterAlt(localctx, 1);
            {
              this.state = 464;
              this.match(_SolidityParser.T__37);
              this.state = 466;
              this._errHandler.sync(this);
              _la = this._input.LA(1);
              if (_la === 14 || _la === 25 || (_la - 44 & ~31) === 0 && (1 << _la - 44 & 262209) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 1615069185) !== 0 || _la === 127 || _la === 128) {
                {
                  this.state = 465;
                  this.identifier();
                }
              }
            }
            break;
          case 125:
            this.enterOuterAlt(localctx, 2);
            {
              this.state = 468;
              this.match(_SolidityParser.ConstructorKeyword);
            }
            break;
          case 126:
            this.enterOuterAlt(localctx, 3);
            {
              this.state = 469;
              this.match(_SolidityParser.FallbackKeyword);
            }
            break;
          case 127:
            this.enterOuterAlt(localctx, 4);
            {
              this.state = 470;
              this.match(_SolidityParser.ReceiveKeyword);
            }
            break;
          default:
            throw new A(this);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    returnParameters() {
      let localctx = new ReturnParametersContext(this, this._ctx, this.state);
      this.enterRule(localctx, 52, _SolidityParser.RULE_returnParameters);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 473;
          this.match(_SolidityParser.T__38);
          this.state = 474;
          this.parameterList();
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    modifierList() {
      let localctx = new ModifierListContext(this, this._ctx, this.state);
      this.enterRule(localctx, 54, _SolidityParser.RULE_modifierList);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 486;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          while (_la === 14 || _la === 25 || (_la - 44 & ~31) === 0 && (1 << _la - 44 & 262209) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 2011987971) !== 0 || _la === 127 || _la === 128) {
            {
              this.state = 484;
              this._errHandler.sync(this);
              switch (this._interp.adaptivePredict(this._input, 41, this._ctx)) {
                case 1:
                  {
                    this.state = 476;
                    this.match(_SolidityParser.ExternalKeyword);
                  }
                  break;
                case 2:
                  {
                    this.state = 477;
                    this.match(_SolidityParser.PublicKeyword);
                  }
                  break;
                case 3:
                  {
                    this.state = 478;
                    this.match(_SolidityParser.InternalKeyword);
                  }
                  break;
                case 4:
                  {
                    this.state = 479;
                    this.match(_SolidityParser.PrivateKeyword);
                  }
                  break;
                case 5:
                  {
                    this.state = 480;
                    this.match(_SolidityParser.VirtualKeyword);
                  }
                  break;
                case 6:
                  {
                    this.state = 481;
                    this.stateMutability();
                  }
                  break;
                case 7:
                  {
                    this.state = 482;
                    this.modifierInvocation();
                  }
                  break;
                case 8:
                  {
                    this.state = 483;
                    this.overrideSpecifier();
                  }
                  break;
              }
            }
            this.state = 488;
            this._errHandler.sync(this);
            _la = this._input.LA(1);
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    eventDefinition() {
      let localctx = new EventDefinitionContext(this, this._ctx, this.state);
      this.enterRule(localctx, 56, _SolidityParser.RULE_eventDefinition);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 489;
          this.match(_SolidityParser.T__39);
          this.state = 490;
          this.identifier();
          this.state = 491;
          this.eventParameterList();
          this.state = 493;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 108) {
            {
              this.state = 492;
              this.match(_SolidityParser.AnonymousKeyword);
            }
          }
          this.state = 495;
          this.match(_SolidityParser.T__1);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    enumValue() {
      let localctx = new EnumValueContext(this, this._ctx, this.state);
      this.enterRule(localctx, 58, _SolidityParser.RULE_enumValue);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 497;
          this.identifier();
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    enumDefinition() {
      let localctx = new EnumDefinitionContext(this, this._ctx, this.state);
      this.enterRule(localctx, 60, _SolidityParser.RULE_enumDefinition);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 499;
          this.match(_SolidityParser.T__40);
          this.state = 500;
          this.identifier();
          this.state = 501;
          this.match(_SolidityParser.T__14);
          this.state = 503;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 14 || _la === 25 || (_la - 44 & ~31) === 0 && (1 << _la - 44 & 262209) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 1615069185) !== 0 || _la === 127 || _la === 128) {
            {
              this.state = 502;
              this.enumValue();
            }
          }
          this.state = 509;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          while (_la === 16) {
            {
              {
                this.state = 505;
                this.match(_SolidityParser.T__15);
                this.state = 506;
                this.enumValue();
              }
            }
            this.state = 511;
            this._errHandler.sync(this);
            _la = this._input.LA(1);
          }
          this.state = 512;
          this.match(_SolidityParser.T__16);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    parameterList() {
      let localctx = new ParameterListContext(this, this._ctx, this.state);
      this.enterRule(localctx, 62, _SolidityParser.RULE_parameterList);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 514;
          this.match(_SolidityParser.T__22);
          this.state = 523;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 14 || _la === 25 || (_la - 38 & ~31) === 0 && (1 << _la - 38 & 520098113) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 1615069309) !== 0 || _la === 127 || _la === 128) {
            {
              this.state = 515;
              this.parameter();
              this.state = 520;
              this._errHandler.sync(this);
              _la = this._input.LA(1);
              while (_la === 16) {
                {
                  {
                    this.state = 516;
                    this.match(_SolidityParser.T__15);
                    this.state = 517;
                    this.parameter();
                  }
                }
                this.state = 522;
                this._errHandler.sync(this);
                _la = this._input.LA(1);
              }
            }
          }
          this.state = 525;
          this.match(_SolidityParser.T__23);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    parameter() {
      let localctx = new ParameterContext(this, this._ctx, this.state);
      this.enterRule(localctx, 64, _SolidityParser.RULE_parameter);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 527;
          this.typeName(0);
          this.state = 529;
          this._errHandler.sync(this);
          switch (this._interp.adaptivePredict(this._input, 48, this._ctx)) {
            case 1:
              {
                this.state = 528;
                this.storageLocation();
              }
              break;
          }
          this.state = 532;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 14 || _la === 25 || (_la - 44 & ~31) === 0 && (1 << _la - 44 & 262209) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 1615069185) !== 0 || _la === 127 || _la === 128) {
            {
              this.state = 531;
              this.identifier();
            }
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    eventParameterList() {
      let localctx = new EventParameterListContext(this, this._ctx, this.state);
      this.enterRule(localctx, 66, _SolidityParser.RULE_eventParameterList);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 534;
          this.match(_SolidityParser.T__22);
          this.state = 543;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 14 || _la === 25 || (_la - 38 & ~31) === 0 && (1 << _la - 38 & 520098113) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 1615069309) !== 0 || _la === 127 || _la === 128) {
            {
              this.state = 535;
              this.eventParameter();
              this.state = 540;
              this._errHandler.sync(this);
              _la = this._input.LA(1);
              while (_la === 16) {
                {
                  {
                    this.state = 536;
                    this.match(_SolidityParser.T__15);
                    this.state = 537;
                    this.eventParameter();
                  }
                }
                this.state = 542;
                this._errHandler.sync(this);
                _la = this._input.LA(1);
              }
            }
          }
          this.state = 545;
          this.match(_SolidityParser.T__23);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    eventParameter() {
      let localctx = new EventParameterContext(this, this._ctx, this.state);
      this.enterRule(localctx, 68, _SolidityParser.RULE_eventParameter);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 547;
          this.typeName(0);
          this.state = 549;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 115) {
            {
              this.state = 548;
              this.match(_SolidityParser.IndexedKeyword);
            }
          }
          this.state = 552;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 14 || _la === 25 || (_la - 44 & ~31) === 0 && (1 << _la - 44 & 262209) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 1615069185) !== 0 || _la === 127 || _la === 128) {
            {
              this.state = 551;
              this.identifier();
            }
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    functionTypeParameterList() {
      let localctx = new FunctionTypeParameterListContext(this, this._ctx, this.state);
      this.enterRule(localctx, 70, _SolidityParser.RULE_functionTypeParameterList);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 554;
          this.match(_SolidityParser.T__22);
          this.state = 563;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 14 || _la === 25 || (_la - 38 & ~31) === 0 && (1 << _la - 38 & 520098113) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 1615069309) !== 0 || _la === 127 || _la === 128) {
            {
              this.state = 555;
              this.functionTypeParameter();
              this.state = 560;
              this._errHandler.sync(this);
              _la = this._input.LA(1);
              while (_la === 16) {
                {
                  {
                    this.state = 556;
                    this.match(_SolidityParser.T__15);
                    this.state = 557;
                    this.functionTypeParameter();
                  }
                }
                this.state = 562;
                this._errHandler.sync(this);
                _la = this._input.LA(1);
              }
            }
          }
          this.state = 565;
          this.match(_SolidityParser.T__23);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    functionTypeParameter() {
      let localctx = new FunctionTypeParameterContext(this, this._ctx, this.state);
      this.enterRule(localctx, 72, _SolidityParser.RULE_functionTypeParameter);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 567;
          this.typeName(0);
          this.state = 569;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if ((_la - 48 & ~31) === 0 && (1 << _la - 48 & 7) !== 0) {
            {
              this.state = 568;
              this.storageLocation();
            }
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    variableDeclaration() {
      let localctx = new VariableDeclarationContext(this, this._ctx, this.state);
      this.enterRule(localctx, 74, _SolidityParser.RULE_variableDeclaration);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 571;
          this.typeName(0);
          this.state = 573;
          this._errHandler.sync(this);
          switch (this._interp.adaptivePredict(this._input, 57, this._ctx)) {
            case 1:
              {
                this.state = 572;
                this.storageLocation();
              }
              break;
          }
          this.state = 575;
          this.identifier();
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    typeName(_p) {
      if (_p === void 0) {
        _p = 0;
      }
      let _parentctx = this._ctx;
      let _parentState = this.state;
      let localctx = new TypeNameContext(this, this._ctx, _parentState);
      let _prevctx = localctx;
      let _startState = 76;
      this.enterRecursionRule(localctx, 76, _SolidityParser.RULE_typeName, _p);
      let _la;
      try {
        let _alt;
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 584;
          this._errHandler.sync(this);
          switch (this._interp.adaptivePredict(this._input, 58, this._ctx)) {
            case 1:
              {
                this.state = 578;
                this.elementaryTypeName();
              }
              break;
            case 2:
              {
                this.state = 579;
                this.userDefinedTypeName();
              }
              break;
            case 3:
              {
                this.state = 580;
                this.mapping();
              }
              break;
            case 4:
              {
                this.state = 581;
                this.functionTypeName();
              }
              break;
            case 5:
              {
                this.state = 582;
                this.match(_SolidityParser.T__43);
                this.state = 583;
                this.match(_SolidityParser.PayableKeyword);
              }
              break;
          }
          this._ctx.stop = this._input.LT(-1);
          this.state = 594;
          this._errHandler.sync(this);
          _alt = this._interp.adaptivePredict(this._input, 60, this._ctx);
          while (_alt !== 2 && _alt !== i.INVALID_ALT_NUMBER) {
            if (_alt === 1) {
              if (this._parseListeners != null) {
                this.triggerExitRuleEvent();
              }
              _prevctx = localctx;
              {
                {
                  localctx = new TypeNameContext(this, _parentctx, _parentState);
                  this.pushNewRecursionContext(localctx, _startState, _SolidityParser.RULE_typeName);
                  this.state = 586;
                  if (!this.precpred(this._ctx, 3)) {
                    throw this.createFailedPredicateException("this.precpred(this._ctx, 3)");
                  }
                  this.state = 587;
                  this.match(_SolidityParser.T__41);
                  this.state = 589;
                  this._errHandler.sync(this);
                  _la = this._input.LA(1);
                  if ((_la & ~31) === 0 && (1 << _la & 3263184960) !== 0 || (_la - 38 & ~31) === 0 && (1 << _la - 38 & 4278194513) !== 0 || (_la - 71 & ~31) === 0 && (1 << _la - 71 & 4244635651) !== 0 || (_la - 103 & ~31) === 0 && (1 << _la - 103 & 124273675) !== 0) {
                    {
                      this.state = 588;
                      this.expression(0);
                    }
                  }
                  this.state = 591;
                  this.match(_SolidityParser.T__42);
                }
              }
            }
            this.state = 596;
            this._errHandler.sync(this);
            _alt = this._interp.adaptivePredict(this._input, 60, this._ctx);
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.unrollRecursionContexts(_parentctx);
      }
      return localctx;
    }
    userDefinedTypeName() {
      let localctx = new UserDefinedTypeNameContext(this, this._ctx, this.state);
      this.enterRule(localctx, 78, _SolidityParser.RULE_userDefinedTypeName);
      try {
        let _alt;
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 597;
          this.identifier();
          this.state = 602;
          this._errHandler.sync(this);
          _alt = this._interp.adaptivePredict(this._input, 61, this._ctx);
          while (_alt !== 2 && _alt !== i.INVALID_ALT_NUMBER) {
            if (_alt === 1) {
              {
                {
                  this.state = 598;
                  this.match(_SolidityParser.T__44);
                  this.state = 599;
                  this.identifier();
                }
              }
            }
            this.state = 604;
            this._errHandler.sync(this);
            _alt = this._interp.adaptivePredict(this._input, 61, this._ctx);
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    mappingKey() {
      let localctx = new MappingKeyContext(this, this._ctx, this.state);
      this.enterRule(localctx, 80, _SolidityParser.RULE_mappingKey);
      try {
        this.state = 607;
        this._errHandler.sync(this);
        switch (this._interp.adaptivePredict(this._input, 62, this._ctx)) {
          case 1:
            this.enterOuterAlt(localctx, 1);
            {
              this.state = 605;
              this.elementaryTypeName();
            }
            break;
          case 2:
            this.enterOuterAlt(localctx, 2);
            {
              this.state = 606;
              this.userDefinedTypeName();
            }
            break;
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    mapping() {
      let localctx = new MappingContext(this, this._ctx, this.state);
      this.enterRule(localctx, 82, _SolidityParser.RULE_mapping);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 609;
          this.match(_SolidityParser.T__45);
          this.state = 610;
          this.match(_SolidityParser.T__22);
          this.state = 611;
          this.mappingKey();
          this.state = 613;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 14 || _la === 25 || (_la - 44 & ~31) === 0 && (1 << _la - 44 & 262209) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 1615069185) !== 0 || _la === 127 || _la === 128) {
            {
              this.state = 612;
              this.mappingKeyName();
            }
          }
          this.state = 615;
          this.match(_SolidityParser.T__46);
          this.state = 616;
          this.typeName(0);
          this.state = 618;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 14 || _la === 25 || (_la - 44 & ~31) === 0 && (1 << _la - 44 & 262209) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 1615069185) !== 0 || _la === 127 || _la === 128) {
            {
              this.state = 617;
              this.mappingValueName();
            }
          }
          this.state = 620;
          this.match(_SolidityParser.T__23);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    mappingKeyName() {
      let localctx = new MappingKeyNameContext(this, this._ctx, this.state);
      this.enterRule(localctx, 84, _SolidityParser.RULE_mappingKeyName);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 622;
          this.identifier();
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    mappingValueName() {
      let localctx = new MappingValueNameContext(this, this._ctx, this.state);
      this.enterRule(localctx, 86, _SolidityParser.RULE_mappingValueName);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 624;
          this.identifier();
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    functionTypeName() {
      let localctx = new FunctionTypeNameContext(this, this._ctx, this.state);
      this.enterRule(localctx, 88, _SolidityParser.RULE_functionTypeName);
      try {
        let _alt;
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 626;
          this.match(_SolidityParser.T__37);
          this.state = 627;
          this.functionTypeParameterList();
          this.state = 633;
          this._errHandler.sync(this);
          _alt = this._interp.adaptivePredict(this._input, 66, this._ctx);
          while (_alt !== 2 && _alt !== i.INVALID_ALT_NUMBER) {
            if (_alt === 1) {
              {
                this.state = 631;
                this._errHandler.sync(this);
                switch (this._input.LA(1)) {
                  case 116:
                    {
                      this.state = 628;
                      this.match(_SolidityParser.InternalKeyword);
                    }
                    break;
                  case 114:
                    {
                      this.state = 629;
                      this.match(_SolidityParser.ExternalKeyword);
                    }
                    break;
                  case 110:
                  case 117:
                  case 121:
                  case 123:
                    {
                      this.state = 630;
                      this.stateMutability();
                    }
                    break;
                  default:
                    throw new A(this);
                }
              }
            }
            this.state = 635;
            this._errHandler.sync(this);
            _alt = this._interp.adaptivePredict(this._input, 66, this._ctx);
          }
          this.state = 638;
          this._errHandler.sync(this);
          switch (this._interp.adaptivePredict(this._input, 67, this._ctx)) {
            case 1:
              {
                this.state = 636;
                this.match(_SolidityParser.T__38);
                this.state = 637;
                this.functionTypeParameterList();
              }
              break;
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    storageLocation() {
      let localctx = new StorageLocationContext(this, this._ctx, this.state);
      this.enterRule(localctx, 90, _SolidityParser.RULE_storageLocation);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 640;
          _la = this._input.LA(1);
          if (!((_la - 48 & ~31) === 0 && (1 << _la - 48 & 7) !== 0)) {
            this._errHandler.recoverInline(this);
          } else {
            this._errHandler.reportMatch(this);
            this.consume();
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    stateMutability() {
      let localctx = new StateMutabilityContext(this, this._ctx, this.state);
      this.enterRule(localctx, 92, _SolidityParser.RULE_stateMutability);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 642;
          _la = this._input.LA(1);
          if (!((_la - 110 & ~31) === 0 && (1 << _la - 110 & 10369) !== 0)) {
            this._errHandler.recoverInline(this);
          } else {
            this._errHandler.reportMatch(this);
            this.consume();
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    block() {
      let localctx = new BlockContext(this, this._ctx, this.state);
      this.enterRule(localctx, 94, _SolidityParser.RULE_block);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 644;
          this.match(_SolidityParser.T__14);
          this.state = 648;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          while ((_la & ~31) === 0 && (1 << _la & 3397435456) !== 0 || (_la - 38 & ~31) === 0 && (1 << _la - 38 & 4294881617) !== 0 || (_la - 71 & ~31) === 0 && (1 << _la - 71 & 4244635651) !== 0 || (_la - 103 & ~31) === 0 && (1 << _la - 103 & 124274251) !== 0) {
            {
              {
                this.state = 645;
                this.statement();
              }
            }
            this.state = 650;
            this._errHandler.sync(this);
            _la = this._input.LA(1);
          }
          this.state = 651;
          this.match(_SolidityParser.T__16);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    statement() {
      let localctx = new StatementContext(this, this._ctx, this.state);
      this.enterRule(localctx, 96, _SolidityParser.RULE_statement);
      try {
        this.state = 668;
        this._errHandler.sync(this);
        switch (this._interp.adaptivePredict(this._input, 69, this._ctx)) {
          case 1:
            this.enterOuterAlt(localctx, 1);
            {
              this.state = 653;
              this.ifStatement();
            }
            break;
          case 2:
            this.enterOuterAlt(localctx, 2);
            {
              this.state = 654;
              this.tryStatement();
            }
            break;
          case 3:
            this.enterOuterAlt(localctx, 3);
            {
              this.state = 655;
              this.whileStatement();
            }
            break;
          case 4:
            this.enterOuterAlt(localctx, 4);
            {
              this.state = 656;
              this.forStatement();
            }
            break;
          case 5:
            this.enterOuterAlt(localctx, 5);
            {
              this.state = 657;
              this.block();
            }
            break;
          case 6:
            this.enterOuterAlt(localctx, 6);
            {
              this.state = 658;
              this.inlineAssemblyStatement();
            }
            break;
          case 7:
            this.enterOuterAlt(localctx, 7);
            {
              this.state = 659;
              this.doWhileStatement();
            }
            break;
          case 8:
            this.enterOuterAlt(localctx, 8);
            {
              this.state = 660;
              this.continueStatement();
            }
            break;
          case 9:
            this.enterOuterAlt(localctx, 9);
            {
              this.state = 661;
              this.breakStatement();
            }
            break;
          case 10:
            this.enterOuterAlt(localctx, 10);
            {
              this.state = 662;
              this.returnStatement();
            }
            break;
          case 11:
            this.enterOuterAlt(localctx, 11);
            {
              this.state = 663;
              this.throwStatement();
            }
            break;
          case 12:
            this.enterOuterAlt(localctx, 12);
            {
              this.state = 664;
              this.emitStatement();
            }
            break;
          case 13:
            this.enterOuterAlt(localctx, 13);
            {
              this.state = 665;
              this.simpleStatement();
            }
            break;
          case 14:
            this.enterOuterAlt(localctx, 14);
            {
              this.state = 666;
              this.uncheckedStatement();
            }
            break;
          case 15:
            this.enterOuterAlt(localctx, 15);
            {
              this.state = 667;
              this.revertStatement();
            }
            break;
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    expressionStatement() {
      let localctx = new ExpressionStatementContext(this, this._ctx, this.state);
      this.enterRule(localctx, 98, _SolidityParser.RULE_expressionStatement);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 670;
          this.expression(0);
          this.state = 671;
          this.match(_SolidityParser.T__1);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    ifStatement() {
      let localctx = new IfStatementContext(this, this._ctx, this.state);
      this.enterRule(localctx, 100, _SolidityParser.RULE_ifStatement);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 673;
          this.match(_SolidityParser.T__50);
          this.state = 674;
          this.match(_SolidityParser.T__22);
          this.state = 675;
          this.expression(0);
          this.state = 676;
          this.match(_SolidityParser.T__23);
          this.state = 677;
          this.statement();
          this.state = 680;
          this._errHandler.sync(this);
          switch (this._interp.adaptivePredict(this._input, 70, this._ctx)) {
            case 1:
              {
                this.state = 678;
                this.match(_SolidityParser.T__51);
                this.state = 679;
                this.statement();
              }
              break;
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    tryStatement() {
      let localctx = new TryStatementContext(this, this._ctx, this.state);
      this.enterRule(localctx, 102, _SolidityParser.RULE_tryStatement);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 682;
          this.match(_SolidityParser.T__52);
          this.state = 683;
          this.expression(0);
          this.state = 685;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 39) {
            {
              this.state = 684;
              this.returnParameters();
            }
          }
          this.state = 687;
          this.block();
          this.state = 689;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          do {
            {
              {
                this.state = 688;
                this.catchClause();
              }
            }
            this.state = 691;
            this._errHandler.sync(this);
            _la = this._input.LA(1);
          } while (_la === 54);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    catchClause() {
      let localctx = new CatchClauseContext(this, this._ctx, this.state);
      this.enterRule(localctx, 104, _SolidityParser.RULE_catchClause);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 693;
          this.match(_SolidityParser.T__53);
          this.state = 698;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if ((_la & ~31) === 0 && (1 << _la & 41959424) !== 0 || (_la - 44 & ~31) === 0 && (1 << _la - 44 & 262209) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 1615069185) !== 0 || _la === 127 || _la === 128) {
            {
              this.state = 695;
              this._errHandler.sync(this);
              _la = this._input.LA(1);
              if (_la === 14 || _la === 25 || (_la - 44 & ~31) === 0 && (1 << _la - 44 & 262209) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 1615069185) !== 0 || _la === 127 || _la === 128) {
                {
                  this.state = 694;
                  this.identifier();
                }
              }
              this.state = 697;
              this.parameterList();
            }
          }
          this.state = 700;
          this.block();
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    whileStatement() {
      let localctx = new WhileStatementContext(this, this._ctx, this.state);
      this.enterRule(localctx, 106, _SolidityParser.RULE_whileStatement);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 702;
          this.match(_SolidityParser.T__54);
          this.state = 703;
          this.match(_SolidityParser.T__22);
          this.state = 704;
          this.expression(0);
          this.state = 705;
          this.match(_SolidityParser.T__23);
          this.state = 706;
          this.statement();
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    simpleStatement() {
      let localctx = new SimpleStatementContext(this, this._ctx, this.state);
      this.enterRule(localctx, 108, _SolidityParser.RULE_simpleStatement);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 710;
          this._errHandler.sync(this);
          switch (this._interp.adaptivePredict(this._input, 75, this._ctx)) {
            case 1:
              {
                this.state = 708;
                this.variableDeclarationStatement();
              }
              break;
            case 2:
              {
                this.state = 709;
                this.expressionStatement();
              }
              break;
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    uncheckedStatement() {
      let localctx = new UncheckedStatementContext(this, this._ctx, this.state);
      this.enterRule(localctx, 110, _SolidityParser.RULE_uncheckedStatement);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 712;
          this.match(_SolidityParser.T__55);
          this.state = 713;
          this.block();
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    forStatement() {
      let localctx = new ForStatementContext(this, this._ctx, this.state);
      this.enterRule(localctx, 112, _SolidityParser.RULE_forStatement);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 715;
          this.match(_SolidityParser.T__26);
          this.state = 716;
          this.match(_SolidityParser.T__22);
          this.state = 719;
          this._errHandler.sync(this);
          switch (this._input.LA(1)) {
            case 6:
            case 14:
            case 23:
            case 25:
            case 30:
            case 31:
            case 38:
            case 42:
            case 44:
            case 46:
            case 50:
            case 62:
            case 63:
            case 64:
            case 65:
            case 66:
            case 67:
            case 68:
            case 69:
            case 71:
            case 72:
            case 95:
            case 97:
            case 98:
            case 99:
            case 100:
            case 101:
            case 102:
            case 103:
            case 104:
            case 106:
            case 113:
            case 117:
            case 122:
            case 124:
            case 125:
            case 127:
            case 128:
            case 129:
              {
                this.state = 717;
                this.simpleStatement();
              }
              break;
            case 2:
              {
                this.state = 718;
                this.match(_SolidityParser.T__1);
              }
              break;
            default:
              throw new A(this);
          }
          this.state = 723;
          this._errHandler.sync(this);
          switch (this._input.LA(1)) {
            case 6:
            case 14:
            case 23:
            case 25:
            case 30:
            case 31:
            case 38:
            case 42:
            case 44:
            case 46:
            case 50:
            case 62:
            case 63:
            case 64:
            case 65:
            case 66:
            case 67:
            case 68:
            case 69:
            case 71:
            case 72:
            case 95:
            case 97:
            case 98:
            case 99:
            case 100:
            case 101:
            case 102:
            case 103:
            case 104:
            case 106:
            case 113:
            case 117:
            case 122:
            case 124:
            case 125:
            case 127:
            case 128:
            case 129:
              {
                this.state = 721;
                this.expressionStatement();
              }
              break;
            case 2:
              {
                this.state = 722;
                this.match(_SolidityParser.T__1);
              }
              break;
            default:
              throw new A(this);
          }
          this.state = 726;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if ((_la & ~31) === 0 && (1 << _la & 3263184960) !== 0 || (_la - 38 & ~31) === 0 && (1 << _la - 38 & 4278194513) !== 0 || (_la - 71 & ~31) === 0 && (1 << _la - 71 & 4244635651) !== 0 || (_la - 103 & ~31) === 0 && (1 << _la - 103 & 124273675) !== 0) {
            {
              this.state = 725;
              this.expression(0);
            }
          }
          this.state = 728;
          this.match(_SolidityParser.T__23);
          this.state = 729;
          this.statement();
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    inlineAssemblyStatement() {
      let localctx = new InlineAssemblyStatementContext(this, this._ctx, this.state);
      this.enterRule(localctx, 114, _SolidityParser.RULE_inlineAssemblyStatement);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 731;
          this.match(_SolidityParser.T__56);
          this.state = 733;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 129) {
            {
              this.state = 732;
              this.match(_SolidityParser.StringLiteralFragment);
            }
          }
          this.state = 739;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 23) {
            {
              this.state = 735;
              this.match(_SolidityParser.T__22);
              this.state = 736;
              this.inlineAssemblyStatementFlag();
              this.state = 737;
              this.match(_SolidityParser.T__23);
            }
          }
          this.state = 741;
          this.assemblyBlock();
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    inlineAssemblyStatementFlag() {
      let localctx = new InlineAssemblyStatementFlagContext(this, this._ctx, this.state);
      this.enterRule(localctx, 116, _SolidityParser.RULE_inlineAssemblyStatementFlag);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 743;
          this.stringLiteral();
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    doWhileStatement() {
      let localctx = new DoWhileStatementContext(this, this._ctx, this.state);
      this.enterRule(localctx, 118, _SolidityParser.RULE_doWhileStatement);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 745;
          this.match(_SolidityParser.T__57);
          this.state = 746;
          this.statement();
          this.state = 747;
          this.match(_SolidityParser.T__54);
          this.state = 748;
          this.match(_SolidityParser.T__22);
          this.state = 749;
          this.expression(0);
          this.state = 750;
          this.match(_SolidityParser.T__23);
          this.state = 751;
          this.match(_SolidityParser.T__1);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    continueStatement() {
      let localctx = new ContinueStatementContext(this, this._ctx, this.state);
      this.enterRule(localctx, 120, _SolidityParser.RULE_continueStatement);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 753;
          this.match(_SolidityParser.ContinueKeyword);
          this.state = 754;
          this.match(_SolidityParser.T__1);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    breakStatement() {
      let localctx = new BreakStatementContext(this, this._ctx, this.state);
      this.enterRule(localctx, 122, _SolidityParser.RULE_breakStatement);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 756;
          this.match(_SolidityParser.BreakKeyword);
          this.state = 757;
          this.match(_SolidityParser.T__1);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    returnStatement() {
      let localctx = new ReturnStatementContext(this, this._ctx, this.state);
      this.enterRule(localctx, 124, _SolidityParser.RULE_returnStatement);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 759;
          this.match(_SolidityParser.T__58);
          this.state = 761;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if ((_la & ~31) === 0 && (1 << _la & 3263184960) !== 0 || (_la - 38 & ~31) === 0 && (1 << _la - 38 & 4278194513) !== 0 || (_la - 71 & ~31) === 0 && (1 << _la - 71 & 4244635651) !== 0 || (_la - 103 & ~31) === 0 && (1 << _la - 103 & 124273675) !== 0) {
            {
              this.state = 760;
              this.expression(0);
            }
          }
          this.state = 763;
          this.match(_SolidityParser.T__1);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    throwStatement() {
      let localctx = new ThrowStatementContext(this, this._ctx, this.state);
      this.enterRule(localctx, 126, _SolidityParser.RULE_throwStatement);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 765;
          this.match(_SolidityParser.T__59);
          this.state = 766;
          this.match(_SolidityParser.T__1);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    emitStatement() {
      let localctx = new EmitStatementContext(this, this._ctx, this.state);
      this.enterRule(localctx, 128, _SolidityParser.RULE_emitStatement);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 768;
          this.match(_SolidityParser.T__60);
          this.state = 769;
          this.functionCall();
          this.state = 770;
          this.match(_SolidityParser.T__1);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    revertStatement() {
      let localctx = new RevertStatementContext(this, this._ctx, this.state);
      this.enterRule(localctx, 130, _SolidityParser.RULE_revertStatement);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 772;
          this.match(_SolidityParser.T__61);
          this.state = 773;
          this.functionCall();
          this.state = 774;
          this.match(_SolidityParser.T__1);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    variableDeclarationStatement() {
      let localctx = new VariableDeclarationStatementContext(this, this._ctx, this.state);
      this.enterRule(localctx, 132, _SolidityParser.RULE_variableDeclarationStatement);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 783;
          this._errHandler.sync(this);
          switch (this._interp.adaptivePredict(this._input, 82, this._ctx)) {
            case 1:
              {
                this.state = 776;
                this.match(_SolidityParser.T__62);
                this.state = 777;
                this.identifierList();
              }
              break;
            case 2:
              {
                this.state = 778;
                this.variableDeclaration();
              }
              break;
            case 3:
              {
                this.state = 779;
                this.match(_SolidityParser.T__22);
                this.state = 780;
                this.variableDeclarationList();
                this.state = 781;
                this.match(_SolidityParser.T__23);
              }
              break;
          }
          this.state = 787;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 11) {
            {
              this.state = 785;
              this.match(_SolidityParser.T__10);
              this.state = 786;
              this.expression(0);
            }
          }
          this.state = 789;
          this.match(_SolidityParser.T__1);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    variableDeclarationList() {
      let localctx = new VariableDeclarationListContext(this, this._ctx, this.state);
      this.enterRule(localctx, 134, _SolidityParser.RULE_variableDeclarationList);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 792;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 14 || _la === 25 || (_la - 38 & ~31) === 0 && (1 << _la - 38 & 520098113) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 1615069309) !== 0 || _la === 127 || _la === 128) {
            {
              this.state = 791;
              this.variableDeclaration();
            }
          }
          this.state = 800;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          while (_la === 16) {
            {
              {
                this.state = 794;
                this.match(_SolidityParser.T__15);
                this.state = 796;
                this._errHandler.sync(this);
                _la = this._input.LA(1);
                if (_la === 14 || _la === 25 || (_la - 38 & ~31) === 0 && (1 << _la - 38 & 520098113) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 1615069309) !== 0 || _la === 127 || _la === 128) {
                  {
                    this.state = 795;
                    this.variableDeclaration();
                  }
                }
              }
            }
            this.state = 802;
            this._errHandler.sync(this);
            _la = this._input.LA(1);
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    identifierList() {
      let localctx = new IdentifierListContext(this, this._ctx, this.state);
      this.enterRule(localctx, 136, _SolidityParser.RULE_identifierList);
      let _la;
      try {
        let _alt;
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 803;
          this.match(_SolidityParser.T__22);
          this.state = 810;
          this._errHandler.sync(this);
          _alt = this._interp.adaptivePredict(this._input, 88, this._ctx);
          while (_alt !== 2 && _alt !== i.INVALID_ALT_NUMBER) {
            if (_alt === 1) {
              {
                {
                  this.state = 805;
                  this._errHandler.sync(this);
                  _la = this._input.LA(1);
                  if (_la === 14 || _la === 25 || (_la - 44 & ~31) === 0 && (1 << _la - 44 & 262209) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 1615069185) !== 0 || _la === 127 || _la === 128) {
                    {
                      this.state = 804;
                      this.identifier();
                    }
                  }
                  this.state = 807;
                  this.match(_SolidityParser.T__15);
                }
              }
            }
            this.state = 812;
            this._errHandler.sync(this);
            _alt = this._interp.adaptivePredict(this._input, 88, this._ctx);
          }
          this.state = 814;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 14 || _la === 25 || (_la - 44 & ~31) === 0 && (1 << _la - 44 & 262209) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 1615069185) !== 0 || _la === 127 || _la === 128) {
            {
              this.state = 813;
              this.identifier();
            }
          }
          this.state = 816;
          this.match(_SolidityParser.T__23);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    elementaryTypeName() {
      let localctx = new ElementaryTypeNameContext(this, this._ctx, this.state);
      this.enterRule(localctx, 138, _SolidityParser.RULE_elementaryTypeName);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 818;
          _la = this._input.LA(1);
          if (!((_la - 44 & ~31) === 0 && (1 << _la - 44 & 7864321) !== 0 || (_la - 97 & ~31) === 0 && (1 << _la - 97 & 31) !== 0)) {
            this._errHandler.recoverInline(this);
          } else {
            this._errHandler.reportMatch(this);
            this.consume();
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    expression(_p) {
      if (_p === void 0) {
        _p = 0;
      }
      let _parentctx = this._ctx;
      let _parentState = this.state;
      let localctx = new ExpressionContext(this, this._ctx, _parentState);
      let _prevctx = localctx;
      let _startState = 140;
      this.enterRecursionRule(localctx, 140, _SolidityParser.RULE_expression, _p);
      let _la;
      try {
        let _alt;
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 838;
          this._errHandler.sync(this);
          switch (this._interp.adaptivePredict(this._input, 90, this._ctx)) {
            case 1:
              {
                this.state = 821;
                this.match(_SolidityParser.T__68);
                this.state = 822;
                this.typeName(0);
              }
              break;
            case 2:
              {
                this.state = 823;
                this.match(_SolidityParser.T__22);
                this.state = 824;
                this.expression(0);
                this.state = 825;
                this.match(_SolidityParser.T__23);
              }
              break;
            case 3:
              {
                this.state = 827;
                _la = this._input.LA(1);
                if (!(_la === 67 || _la === 68)) {
                  this._errHandler.recoverInline(this);
                } else {
                  this._errHandler.reportMatch(this);
                  this.consume();
                }
                this.state = 828;
                this.expression(19);
              }
              break;
            case 4:
              {
                this.state = 829;
                _la = this._input.LA(1);
                if (!(_la === 30 || _la === 31)) {
                  this._errHandler.recoverInline(this);
                } else {
                  this._errHandler.reportMatch(this);
                  this.consume();
                }
                this.state = 830;
                this.expression(18);
              }
              break;
            case 5:
              {
                this.state = 831;
                this.match(_SolidityParser.T__70);
                this.state = 832;
                this.expression(17);
              }
              break;
            case 6:
              {
                this.state = 833;
                this.match(_SolidityParser.T__71);
                this.state = 834;
                this.expression(16);
              }
              break;
            case 7:
              {
                this.state = 835;
                this.match(_SolidityParser.T__5);
                this.state = 836;
                this.expression(15);
              }
              break;
            case 8:
              {
                this.state = 837;
                this.primaryExpression();
              }
              break;
          }
          this._ctx.stop = this._input.LT(-1);
          this.state = 914;
          this._errHandler.sync(this);
          _alt = this._interp.adaptivePredict(this._input, 94, this._ctx);
          while (_alt !== 2 && _alt !== i.INVALID_ALT_NUMBER) {
            if (_alt === 1) {
              if (this._parseListeners != null) {
                this.triggerExitRuleEvent();
              }
              _prevctx = localctx;
              {
                this.state = 912;
                this._errHandler.sync(this);
                switch (this._interp.adaptivePredict(this._input, 93, this._ctx)) {
                  case 1:
                    {
                      localctx = new ExpressionContext(this, _parentctx, _parentState);
                      this.pushNewRecursionContext(localctx, _startState, _SolidityParser.RULE_expression);
                      this.state = 840;
                      if (!this.precpred(this._ctx, 14)) {
                        throw this.createFailedPredicateException("this.precpred(this._ctx, 14)");
                      }
                      this.state = 841;
                      this.match(_SolidityParser.T__72);
                      this.state = 842;
                      this.expression(14);
                    }
                    break;
                  case 2:
                    {
                      localctx = new ExpressionContext(this, _parentctx, _parentState);
                      this.pushNewRecursionContext(localctx, _startState, _SolidityParser.RULE_expression);
                      this.state = 843;
                      if (!this.precpred(this._ctx, 13)) {
                        throw this.createFailedPredicateException("this.precpred(this._ctx, 13)");
                      }
                      this.state = 844;
                      _la = this._input.LA(1);
                      if (!((_la - 3 & ~31) === 0 && (1 << _la - 3 & 1610612737) !== 0)) {
                        this._errHandler.recoverInline(this);
                      } else {
                        this._errHandler.reportMatch(this);
                        this.consume();
                      }
                      this.state = 845;
                      this.expression(14);
                    }
                    break;
                  case 3:
                    {
                      localctx = new ExpressionContext(this, _parentctx, _parentState);
                      this.pushNewRecursionContext(localctx, _startState, _SolidityParser.RULE_expression);
                      this.state = 846;
                      if (!this.precpred(this._ctx, 12)) {
                        throw this.createFailedPredicateException("this.precpred(this._ctx, 12)");
                      }
                      this.state = 847;
                      _la = this._input.LA(1);
                      if (!(_la === 30 || _la === 31)) {
                        this._errHandler.recoverInline(this);
                      } else {
                        this._errHandler.reportMatch(this);
                        this.consume();
                      }
                      this.state = 848;
                      this.expression(13);
                    }
                    break;
                  case 4:
                    {
                      localctx = new ExpressionContext(this, _parentctx, _parentState);
                      this.pushNewRecursionContext(localctx, _startState, _SolidityParser.RULE_expression);
                      this.state = 849;
                      if (!this.precpred(this._ctx, 11)) {
                        throw this.createFailedPredicateException("this.precpred(this._ctx, 11)");
                      }
                      this.state = 850;
                      _la = this._input.LA(1);
                      if (!(_la === 74 || _la === 75)) {
                        this._errHandler.recoverInline(this);
                      } else {
                        this._errHandler.reportMatch(this);
                        this.consume();
                      }
                      this.state = 851;
                      this.expression(12);
                    }
                    break;
                  case 5:
                    {
                      localctx = new ExpressionContext(this, _parentctx, _parentState);
                      this.pushNewRecursionContext(localctx, _startState, _SolidityParser.RULE_expression);
                      this.state = 852;
                      if (!this.precpred(this._ctx, 10)) {
                        throw this.createFailedPredicateException("this.precpred(this._ctx, 10)");
                      }
                      this.state = 853;
                      this.match(_SolidityParser.T__28);
                      this.state = 854;
                      this.expression(11);
                    }
                    break;
                  case 6:
                    {
                      localctx = new ExpressionContext(this, _parentctx, _parentState);
                      this.pushNewRecursionContext(localctx, _startState, _SolidityParser.RULE_expression);
                      this.state = 855;
                      if (!this.precpred(this._ctx, 9)) {
                        throw this.createFailedPredicateException("this.precpred(this._ctx, 9)");
                      }
                      this.state = 856;
                      this.match(_SolidityParser.T__4);
                      this.state = 857;
                      this.expression(10);
                    }
                    break;
                  case 7:
                    {
                      localctx = new ExpressionContext(this, _parentctx, _parentState);
                      this.pushNewRecursionContext(localctx, _startState, _SolidityParser.RULE_expression);
                      this.state = 858;
                      if (!this.precpred(this._ctx, 8)) {
                        throw this.createFailedPredicateException("this.precpred(this._ctx, 8)");
                      }
                      this.state = 859;
                      this.match(_SolidityParser.T__27);
                      this.state = 860;
                      this.expression(9);
                    }
                    break;
                  case 8:
                    {
                      localctx = new ExpressionContext(this, _parentctx, _parentState);
                      this.pushNewRecursionContext(localctx, _startState, _SolidityParser.RULE_expression);
                      this.state = 861;
                      if (!this.precpred(this._ctx, 7)) {
                        throw this.createFailedPredicateException("this.precpred(this._ctx, 7)");
                      }
                      this.state = 862;
                      _la = this._input.LA(1);
                      if (!((_la & ~31) === 0 && (1 << _la & 1920) !== 0)) {
                        this._errHandler.recoverInline(this);
                      } else {
                        this._errHandler.reportMatch(this);
                        this.consume();
                      }
                      this.state = 863;
                      this.expression(8);
                    }
                    break;
                  case 9:
                    {
                      localctx = new ExpressionContext(this, _parentctx, _parentState);
                      this.pushNewRecursionContext(localctx, _startState, _SolidityParser.RULE_expression);
                      this.state = 864;
                      if (!this.precpred(this._ctx, 6)) {
                        throw this.createFailedPredicateException("this.precpred(this._ctx, 6)");
                      }
                      this.state = 865;
                      _la = this._input.LA(1);
                      if (!(_la === 34 || _la === 35)) {
                        this._errHandler.recoverInline(this);
                      } else {
                        this._errHandler.reportMatch(this);
                        this.consume();
                      }
                      this.state = 866;
                      this.expression(7);
                    }
                    break;
                  case 10:
                    {
                      localctx = new ExpressionContext(this, _parentctx, _parentState);
                      this.pushNewRecursionContext(localctx, _startState, _SolidityParser.RULE_expression);
                      this.state = 867;
                      if (!this.precpred(this._ctx, 5)) {
                        throw this.createFailedPredicateException("this.precpred(this._ctx, 5)");
                      }
                      this.state = 868;
                      this.match(_SolidityParser.T__75);
                      this.state = 869;
                      this.expression(6);
                    }
                    break;
                  case 11:
                    {
                      localctx = new ExpressionContext(this, _parentctx, _parentState);
                      this.pushNewRecursionContext(localctx, _startState, _SolidityParser.RULE_expression);
                      this.state = 870;
                      if (!this.precpred(this._ctx, 4)) {
                        throw this.createFailedPredicateException("this.precpred(this._ctx, 4)");
                      }
                      this.state = 871;
                      this.match(_SolidityParser.T__3);
                      this.state = 872;
                      this.expression(5);
                    }
                    break;
                  case 12:
                    {
                      localctx = new ExpressionContext(this, _parentctx, _parentState);
                      this.pushNewRecursionContext(localctx, _startState, _SolidityParser.RULE_expression);
                      this.state = 873;
                      if (!this.precpred(this._ctx, 3)) {
                        throw this.createFailedPredicateException("this.precpred(this._ctx, 3)");
                      }
                      this.state = 874;
                      this.match(_SolidityParser.T__76);
                      this.state = 875;
                      this.expression(0);
                      this.state = 876;
                      this.match(_SolidityParser.T__69);
                      this.state = 877;
                      this.expression(3);
                    }
                    break;
                  case 13:
                    {
                      localctx = new ExpressionContext(this, _parentctx, _parentState);
                      this.pushNewRecursionContext(localctx, _startState, _SolidityParser.RULE_expression);
                      this.state = 879;
                      if (!this.precpred(this._ctx, 2)) {
                        throw this.createFailedPredicateException("this.precpred(this._ctx, 2)");
                      }
                      this.state = 880;
                      _la = this._input.LA(1);
                      if (!(_la === 11 || (_la - 78 & ~31) === 0 && (1 << _la - 78 & 1023) !== 0)) {
                        this._errHandler.recoverInline(this);
                      } else {
                        this._errHandler.reportMatch(this);
                        this.consume();
                      }
                      this.state = 881;
                      this.expression(3);
                    }
                    break;
                  case 14:
                    {
                      localctx = new ExpressionContext(this, _parentctx, _parentState);
                      this.pushNewRecursionContext(localctx, _startState, _SolidityParser.RULE_expression);
                      this.state = 882;
                      if (!this.precpred(this._ctx, 27)) {
                        throw this.createFailedPredicateException("this.precpred(this._ctx, 27)");
                      }
                      this.state = 883;
                      _la = this._input.LA(1);
                      if (!(_la === 67 || _la === 68)) {
                        this._errHandler.recoverInline(this);
                      } else {
                        this._errHandler.reportMatch(this);
                        this.consume();
                      }
                    }
                    break;
                  case 15:
                    {
                      localctx = new ExpressionContext(this, _parentctx, _parentState);
                      this.pushNewRecursionContext(localctx, _startState, _SolidityParser.RULE_expression);
                      this.state = 884;
                      if (!this.precpred(this._ctx, 25)) {
                        throw this.createFailedPredicateException("this.precpred(this._ctx, 25)");
                      }
                      this.state = 885;
                      this.match(_SolidityParser.T__41);
                      this.state = 886;
                      this.expression(0);
                      this.state = 887;
                      this.match(_SolidityParser.T__42);
                    }
                    break;
                  case 16:
                    {
                      localctx = new ExpressionContext(this, _parentctx, _parentState);
                      this.pushNewRecursionContext(localctx, _startState, _SolidityParser.RULE_expression);
                      this.state = 889;
                      if (!this.precpred(this._ctx, 24)) {
                        throw this.createFailedPredicateException("this.precpred(this._ctx, 24)");
                      }
                      this.state = 890;
                      this.match(_SolidityParser.T__41);
                      this.state = 892;
                      this._errHandler.sync(this);
                      _la = this._input.LA(1);
                      if ((_la & ~31) === 0 && (1 << _la & 3263184960) !== 0 || (_la - 38 & ~31) === 0 && (1 << _la - 38 & 4278194513) !== 0 || (_la - 71 & ~31) === 0 && (1 << _la - 71 & 4244635651) !== 0 || (_la - 103 & ~31) === 0 && (1 << _la - 103 & 124273675) !== 0) {
                        {
                          this.state = 891;
                          this.expression(0);
                        }
                      }
                      this.state = 894;
                      this.match(_SolidityParser.T__69);
                      this.state = 896;
                      this._errHandler.sync(this);
                      _la = this._input.LA(1);
                      if ((_la & ~31) === 0 && (1 << _la & 3263184960) !== 0 || (_la - 38 & ~31) === 0 && (1 << _la - 38 & 4278194513) !== 0 || (_la - 71 & ~31) === 0 && (1 << _la - 71 & 4244635651) !== 0 || (_la - 103 & ~31) === 0 && (1 << _la - 103 & 124273675) !== 0) {
                        {
                          this.state = 895;
                          this.expression(0);
                        }
                      }
                      this.state = 898;
                      this.match(_SolidityParser.T__42);
                    }
                    break;
                  case 17:
                    {
                      localctx = new ExpressionContext(this, _parentctx, _parentState);
                      this.pushNewRecursionContext(localctx, _startState, _SolidityParser.RULE_expression);
                      this.state = 899;
                      if (!this.precpred(this._ctx, 23)) {
                        throw this.createFailedPredicateException("this.precpred(this._ctx, 23)");
                      }
                      this.state = 900;
                      this.match(_SolidityParser.T__44);
                      this.state = 901;
                      this.identifier();
                    }
                    break;
                  case 18:
                    {
                      localctx = new ExpressionContext(this, _parentctx, _parentState);
                      this.pushNewRecursionContext(localctx, _startState, _SolidityParser.RULE_expression);
                      this.state = 902;
                      if (!this.precpred(this._ctx, 22)) {
                        throw this.createFailedPredicateException("this.precpred(this._ctx, 22)");
                      }
                      this.state = 903;
                      this.match(_SolidityParser.T__14);
                      this.state = 904;
                      this.nameValueList();
                      this.state = 905;
                      this.match(_SolidityParser.T__16);
                    }
                    break;
                  case 19:
                    {
                      localctx = new ExpressionContext(this, _parentctx, _parentState);
                      this.pushNewRecursionContext(localctx, _startState, _SolidityParser.RULE_expression);
                      this.state = 907;
                      if (!this.precpred(this._ctx, 21)) {
                        throw this.createFailedPredicateException("this.precpred(this._ctx, 21)");
                      }
                      this.state = 908;
                      this.match(_SolidityParser.T__22);
                      this.state = 909;
                      this.functionCallArguments();
                      this.state = 910;
                      this.match(_SolidityParser.T__23);
                    }
                    break;
                }
              }
            }
            this.state = 916;
            this._errHandler.sync(this);
            _alt = this._interp.adaptivePredict(this._input, 94, this._ctx);
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.unrollRecursionContexts(_parentctx);
      }
      return localctx;
    }
    primaryExpression() {
      let localctx = new PrimaryExpressionContext(this, this._ctx, this.state);
      this.enterRule(localctx, 142, _SolidityParser.RULE_primaryExpression);
      try {
        this.state = 926;
        this._errHandler.sync(this);
        switch (this._interp.adaptivePredict(this._input, 95, this._ctx)) {
          case 1:
            this.enterOuterAlt(localctx, 1);
            {
              this.state = 917;
              this.match(_SolidityParser.BooleanLiteral);
            }
            break;
          case 2:
            this.enterOuterAlt(localctx, 2);
            {
              this.state = 918;
              this.numberLiteral();
            }
            break;
          case 3:
            this.enterOuterAlt(localctx, 3);
            {
              this.state = 919;
              this.hexLiteral();
            }
            break;
          case 4:
            this.enterOuterAlt(localctx, 4);
            {
              this.state = 920;
              this.stringLiteral();
            }
            break;
          case 5:
            this.enterOuterAlt(localctx, 5);
            {
              this.state = 921;
              this.identifier();
            }
            break;
          case 6:
            this.enterOuterAlt(localctx, 6);
            {
              this.state = 922;
              this.match(_SolidityParser.TypeKeyword);
            }
            break;
          case 7:
            this.enterOuterAlt(localctx, 7);
            {
              this.state = 923;
              this.match(_SolidityParser.PayableKeyword);
            }
            break;
          case 8:
            this.enterOuterAlt(localctx, 8);
            {
              this.state = 924;
              this.tupleExpression();
            }
            break;
          case 9:
            this.enterOuterAlt(localctx, 9);
            {
              this.state = 925;
              this.typeName(0);
            }
            break;
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    expressionList() {
      let localctx = new ExpressionListContext(this, this._ctx, this.state);
      this.enterRule(localctx, 144, _SolidityParser.RULE_expressionList);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 928;
          this.expression(0);
          this.state = 933;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          while (_la === 16) {
            {
              {
                this.state = 929;
                this.match(_SolidityParser.T__15);
                this.state = 930;
                this.expression(0);
              }
            }
            this.state = 935;
            this._errHandler.sync(this);
            _la = this._input.LA(1);
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    nameValueList() {
      let localctx = new NameValueListContext(this, this._ctx, this.state);
      this.enterRule(localctx, 146, _SolidityParser.RULE_nameValueList);
      let _la;
      try {
        let _alt;
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 936;
          this.nameValue();
          this.state = 941;
          this._errHandler.sync(this);
          _alt = this._interp.adaptivePredict(this._input, 97, this._ctx);
          while (_alt !== 2 && _alt !== i.INVALID_ALT_NUMBER) {
            if (_alt === 1) {
              {
                {
                  this.state = 937;
                  this.match(_SolidityParser.T__15);
                  this.state = 938;
                  this.nameValue();
                }
              }
            }
            this.state = 943;
            this._errHandler.sync(this);
            _alt = this._interp.adaptivePredict(this._input, 97, this._ctx);
          }
          this.state = 945;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 16) {
            {
              this.state = 944;
              this.match(_SolidityParser.T__15);
            }
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    nameValue() {
      let localctx = new NameValueContext(this, this._ctx, this.state);
      this.enterRule(localctx, 148, _SolidityParser.RULE_nameValue);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 947;
          this.identifier();
          this.state = 948;
          this.match(_SolidityParser.T__69);
          this.state = 949;
          this.expression(0);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    functionCallArguments() {
      let localctx = new FunctionCallArgumentsContext(this, this._ctx, this.state);
      this.enterRule(localctx, 150, _SolidityParser.RULE_functionCallArguments);
      let _la;
      try {
        this.state = 959;
        this._errHandler.sync(this);
        switch (this._input.LA(1)) {
          case 15:
            this.enterOuterAlt(localctx, 1);
            {
              this.state = 951;
              this.match(_SolidityParser.T__14);
              this.state = 953;
              this._errHandler.sync(this);
              _la = this._input.LA(1);
              if (_la === 14 || _la === 25 || (_la - 44 & ~31) === 0 && (1 << _la - 44 & 262209) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 1615069185) !== 0 || _la === 127 || _la === 128) {
                {
                  this.state = 952;
                  this.nameValueList();
                }
              }
              this.state = 955;
              this.match(_SolidityParser.T__16);
            }
            break;
          case 6:
          case 14:
          case 23:
          case 24:
          case 25:
          case 30:
          case 31:
          case 38:
          case 42:
          case 44:
          case 46:
          case 50:
          case 62:
          case 63:
          case 64:
          case 65:
          case 66:
          case 67:
          case 68:
          case 69:
          case 71:
          case 72:
          case 95:
          case 97:
          case 98:
          case 99:
          case 100:
          case 101:
          case 102:
          case 103:
          case 104:
          case 106:
          case 113:
          case 117:
          case 122:
          case 124:
          case 125:
          case 127:
          case 128:
          case 129:
            this.enterOuterAlt(localctx, 2);
            {
              this.state = 957;
              this._errHandler.sync(this);
              _la = this._input.LA(1);
              if ((_la & ~31) === 0 && (1 << _la & 3263184960) !== 0 || (_la - 38 & ~31) === 0 && (1 << _la - 38 & 4278194513) !== 0 || (_la - 71 & ~31) === 0 && (1 << _la - 71 & 4244635651) !== 0 || (_la - 103 & ~31) === 0 && (1 << _la - 103 & 124273675) !== 0) {
                {
                  this.state = 956;
                  this.expressionList();
                }
              }
            }
            break;
          default:
            throw new A(this);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    functionCall() {
      let localctx = new FunctionCallContext(this, this._ctx, this.state);
      this.enterRule(localctx, 152, _SolidityParser.RULE_functionCall);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 961;
          this.expression(0);
          this.state = 962;
          this.match(_SolidityParser.T__22);
          this.state = 963;
          this.functionCallArguments();
          this.state = 964;
          this.match(_SolidityParser.T__23);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    assemblyBlock() {
      let localctx = new AssemblyBlockContext(this, this._ctx, this.state);
      this.enterRule(localctx, 154, _SolidityParser.RULE_assemblyBlock);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 966;
          this.match(_SolidityParser.T__14);
          this.state = 970;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          while ((_la & ~31) === 0 && (1 << _la & 176209920) !== 0 || (_la - 38 & ~31) === 0 && (1 << _la - 38 & 287322177) !== 0 || (_la - 88 & ~31) === 0 && (1 << _la - 88 & 589676681) !== 0 || (_la - 124 & ~31) === 0 && (1 << _la - 124 & 59) !== 0) {
            {
              {
                this.state = 967;
                this.assemblyItem();
              }
            }
            this.state = 972;
            this._errHandler.sync(this);
            _la = this._input.LA(1);
          }
          this.state = 973;
          this.match(_SolidityParser.T__16);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    assemblyItem() {
      let localctx = new AssemblyItemContext(this, this._ctx, this.state);
      this.enterRule(localctx, 156, _SolidityParser.RULE_assemblyItem);
      try {
        this.state = 992;
        this._errHandler.sync(this);
        switch (this._interp.adaptivePredict(this._input, 103, this._ctx)) {
          case 1:
            this.enterOuterAlt(localctx, 1);
            {
              this.state = 975;
              this.identifier();
            }
            break;
          case 2:
            this.enterOuterAlt(localctx, 2);
            {
              this.state = 976;
              this.assemblyBlock();
            }
            break;
          case 3:
            this.enterOuterAlt(localctx, 3);
            {
              this.state = 977;
              this.assemblyExpression();
            }
            break;
          case 4:
            this.enterOuterAlt(localctx, 4);
            {
              this.state = 978;
              this.assemblyLocalDefinition();
            }
            break;
          case 5:
            this.enterOuterAlt(localctx, 5);
            {
              this.state = 979;
              this.assemblyAssignment();
            }
            break;
          case 6:
            this.enterOuterAlt(localctx, 6);
            {
              this.state = 980;
              this.assemblyStackAssignment();
            }
            break;
          case 7:
            this.enterOuterAlt(localctx, 7);
            {
              this.state = 981;
              this.labelDefinition();
            }
            break;
          case 8:
            this.enterOuterAlt(localctx, 8);
            {
              this.state = 982;
              this.assemblySwitch();
            }
            break;
          case 9:
            this.enterOuterAlt(localctx, 9);
            {
              this.state = 983;
              this.assemblyFunctionDefinition();
            }
            break;
          case 10:
            this.enterOuterAlt(localctx, 10);
            {
              this.state = 984;
              this.assemblyFor();
            }
            break;
          case 11:
            this.enterOuterAlt(localctx, 11);
            {
              this.state = 985;
              this.assemblyIf();
            }
            break;
          case 12:
            this.enterOuterAlt(localctx, 12);
            {
              this.state = 986;
              this.match(_SolidityParser.BreakKeyword);
            }
            break;
          case 13:
            this.enterOuterAlt(localctx, 13);
            {
              this.state = 987;
              this.match(_SolidityParser.ContinueKeyword);
            }
            break;
          case 14:
            this.enterOuterAlt(localctx, 14);
            {
              this.state = 988;
              this.match(_SolidityParser.LeaveKeyword);
            }
            break;
          case 15:
            this.enterOuterAlt(localctx, 15);
            {
              this.state = 989;
              this.numberLiteral();
            }
            break;
          case 16:
            this.enterOuterAlt(localctx, 16);
            {
              this.state = 990;
              this.stringLiteral();
            }
            break;
          case 17:
            this.enterOuterAlt(localctx, 17);
            {
              this.state = 991;
              this.hexLiteral();
            }
            break;
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    assemblyExpression() {
      let localctx = new AssemblyExpressionContext(this, this._ctx, this.state);
      this.enterRule(localctx, 158, _SolidityParser.RULE_assemblyExpression);
      try {
        this.state = 997;
        this._errHandler.sync(this);
        switch (this._interp.adaptivePredict(this._input, 104, this._ctx)) {
          case 1:
            this.enterOuterAlt(localctx, 1);
            {
              this.state = 994;
              this.assemblyCall();
            }
            break;
          case 2:
            this.enterOuterAlt(localctx, 2);
            {
              this.state = 995;
              this.assemblyLiteral();
            }
            break;
          case 3:
            this.enterOuterAlt(localctx, 3);
            {
              this.state = 996;
              this.assemblyMember();
            }
            break;
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    assemblyMember() {
      let localctx = new AssemblyMemberContext(this, this._ctx, this.state);
      this.enterRule(localctx, 160, _SolidityParser.RULE_assemblyMember);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 999;
          this.identifier();
          this.state = 1e3;
          this.match(_SolidityParser.T__44);
          this.state = 1001;
          this.identifier();
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    assemblyCall() {
      let localctx = new AssemblyCallContext(this, this._ctx, this.state);
      this.enterRule(localctx, 162, _SolidityParser.RULE_assemblyCall);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 1007;
          this._errHandler.sync(this);
          switch (this._interp.adaptivePredict(this._input, 105, this._ctx)) {
            case 1:
              {
                this.state = 1003;
                this.match(_SolidityParser.T__58);
              }
              break;
            case 2:
              {
                this.state = 1004;
                this.match(_SolidityParser.T__43);
              }
              break;
            case 3:
              {
                this.state = 1005;
                this.match(_SolidityParser.T__65);
              }
              break;
            case 4:
              {
                this.state = 1006;
                this.identifier();
              }
              break;
          }
          this.state = 1021;
          this._errHandler.sync(this);
          switch (this._interp.adaptivePredict(this._input, 108, this._ctx)) {
            case 1:
              {
                this.state = 1009;
                this.match(_SolidityParser.T__22);
                this.state = 1011;
                this._errHandler.sync(this);
                _la = this._input.LA(1);
                if (_la === 14 || _la === 25 || (_la - 44 & ~31) === 0 && (1 << _la - 44 & 4489281) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 1615072129) !== 0 || (_la - 127 & ~31) === 0 && (1 << _la - 127 & 7) !== 0) {
                  {
                    this.state = 1010;
                    this.assemblyExpression();
                  }
                }
                this.state = 1017;
                this._errHandler.sync(this);
                _la = this._input.LA(1);
                while (_la === 16) {
                  {
                    {
                      this.state = 1013;
                      this.match(_SolidityParser.T__15);
                      this.state = 1014;
                      this.assemblyExpression();
                    }
                  }
                  this.state = 1019;
                  this._errHandler.sync(this);
                  _la = this._input.LA(1);
                }
                this.state = 1020;
                this.match(_SolidityParser.T__23);
              }
              break;
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    assemblyLocalDefinition() {
      let localctx = new AssemblyLocalDefinitionContext(this, this._ctx, this.state);
      this.enterRule(localctx, 164, _SolidityParser.RULE_assemblyLocalDefinition);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 1023;
          this.match(_SolidityParser.T__87);
          this.state = 1024;
          this.assemblyIdentifierOrList();
          this.state = 1027;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 89) {
            {
              this.state = 1025;
              this.match(_SolidityParser.T__88);
              this.state = 1026;
              this.assemblyExpression();
            }
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    assemblyAssignment() {
      let localctx = new AssemblyAssignmentContext(this, this._ctx, this.state);
      this.enterRule(localctx, 166, _SolidityParser.RULE_assemblyAssignment);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 1029;
          this.assemblyIdentifierOrList();
          this.state = 1030;
          this.match(_SolidityParser.T__88);
          this.state = 1031;
          this.assemblyExpression();
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    assemblyIdentifierOrList() {
      let localctx = new AssemblyIdentifierOrListContext(this, this._ctx, this.state);
      this.enterRule(localctx, 168, _SolidityParser.RULE_assemblyIdentifierOrList);
      try {
        this.state = 1040;
        this._errHandler.sync(this);
        switch (this._interp.adaptivePredict(this._input, 110, this._ctx)) {
          case 1:
            this.enterOuterAlt(localctx, 1);
            {
              this.state = 1033;
              this.identifier();
            }
            break;
          case 2:
            this.enterOuterAlt(localctx, 2);
            {
              this.state = 1034;
              this.assemblyMember();
            }
            break;
          case 3:
            this.enterOuterAlt(localctx, 3);
            {
              this.state = 1035;
              this.assemblyIdentifierList();
            }
            break;
          case 4:
            this.enterOuterAlt(localctx, 4);
            {
              this.state = 1036;
              this.match(_SolidityParser.T__22);
              this.state = 1037;
              this.assemblyIdentifierList();
              this.state = 1038;
              this.match(_SolidityParser.T__23);
            }
            break;
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    assemblyIdentifierList() {
      let localctx = new AssemblyIdentifierListContext(this, this._ctx, this.state);
      this.enterRule(localctx, 170, _SolidityParser.RULE_assemblyIdentifierList);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 1042;
          this.identifier();
          this.state = 1047;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          while (_la === 16) {
            {
              {
                this.state = 1043;
                this.match(_SolidityParser.T__15);
                this.state = 1044;
                this.identifier();
              }
            }
            this.state = 1049;
            this._errHandler.sync(this);
            _la = this._input.LA(1);
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    assemblyStackAssignment() {
      let localctx = new AssemblyStackAssignmentContext(this, this._ctx, this.state);
      this.enterRule(localctx, 172, _SolidityParser.RULE_assemblyStackAssignment);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 1050;
          this.assemblyExpression();
          this.state = 1051;
          this.match(_SolidityParser.T__89);
          this.state = 1052;
          this.identifier();
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    labelDefinition() {
      let localctx = new LabelDefinitionContext(this, this._ctx, this.state);
      this.enterRule(localctx, 174, _SolidityParser.RULE_labelDefinition);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 1054;
          this.identifier();
          this.state = 1055;
          this.match(_SolidityParser.T__69);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    assemblySwitch() {
      let localctx = new AssemblySwitchContext(this, this._ctx, this.state);
      this.enterRule(localctx, 176, _SolidityParser.RULE_assemblySwitch);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 1057;
          this.match(_SolidityParser.T__90);
          this.state = 1058;
          this.assemblyExpression();
          this.state = 1062;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          while (_la === 92 || _la === 93) {
            {
              {
                this.state = 1059;
                this.assemblyCase();
              }
            }
            this.state = 1064;
            this._errHandler.sync(this);
            _la = this._input.LA(1);
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    assemblyCase() {
      let localctx = new AssemblyCaseContext(this, this._ctx, this.state);
      this.enterRule(localctx, 178, _SolidityParser.RULE_assemblyCase);
      try {
        this.state = 1071;
        this._errHandler.sync(this);
        switch (this._input.LA(1)) {
          case 92:
            this.enterOuterAlt(localctx, 1);
            {
              this.state = 1065;
              this.match(_SolidityParser.T__91);
              this.state = 1066;
              this.assemblyLiteral();
              this.state = 1067;
              this.assemblyBlock();
            }
            break;
          case 93:
            this.enterOuterAlt(localctx, 2);
            {
              this.state = 1069;
              this.match(_SolidityParser.T__92);
              this.state = 1070;
              this.assemblyBlock();
            }
            break;
          default:
            throw new A(this);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    assemblyFunctionDefinition() {
      let localctx = new AssemblyFunctionDefinitionContext(this, this._ctx, this.state);
      this.enterRule(localctx, 180, _SolidityParser.RULE_assemblyFunctionDefinition);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 1073;
          this.match(_SolidityParser.T__37);
          this.state = 1074;
          this.identifier();
          this.state = 1075;
          this.match(_SolidityParser.T__22);
          this.state = 1077;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 14 || _la === 25 || (_la - 44 & ~31) === 0 && (1 << _la - 44 & 262209) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 1615069185) !== 0 || _la === 127 || _la === 128) {
            {
              this.state = 1076;
              this.assemblyIdentifierList();
            }
          }
          this.state = 1079;
          this.match(_SolidityParser.T__23);
          this.state = 1081;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 94) {
            {
              this.state = 1080;
              this.assemblyFunctionReturns();
            }
          }
          this.state = 1083;
          this.assemblyBlock();
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    assemblyFunctionReturns() {
      let localctx = new AssemblyFunctionReturnsContext(this, this._ctx, this.state);
      this.enterRule(localctx, 182, _SolidityParser.RULE_assemblyFunctionReturns);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          {
            this.state = 1085;
            this.match(_SolidityParser.T__93);
            this.state = 1086;
            this.assemblyIdentifierList();
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    assemblyFor() {
      let localctx = new AssemblyForContext(this, this._ctx, this.state);
      this.enterRule(localctx, 184, _SolidityParser.RULE_assemblyFor);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 1088;
          this.match(_SolidityParser.T__26);
          this.state = 1091;
          this._errHandler.sync(this);
          switch (this._input.LA(1)) {
            case 15:
              {
                this.state = 1089;
                this.assemblyBlock();
              }
              break;
            case 14:
            case 25:
            case 44:
            case 50:
            case 59:
            case 62:
            case 66:
            case 95:
            case 102:
            case 103:
            case 104:
            case 106:
            case 113:
            case 117:
            case 124:
            case 125:
            case 127:
            case 128:
            case 129:
              {
                this.state = 1090;
                this.assemblyExpression();
              }
              break;
            default:
              throw new A(this);
          }
          this.state = 1093;
          this.assemblyExpression();
          this.state = 1096;
          this._errHandler.sync(this);
          switch (this._input.LA(1)) {
            case 15:
              {
                this.state = 1094;
                this.assemblyBlock();
              }
              break;
            case 14:
            case 25:
            case 44:
            case 50:
            case 59:
            case 62:
            case 66:
            case 95:
            case 102:
            case 103:
            case 104:
            case 106:
            case 113:
            case 117:
            case 124:
            case 125:
            case 127:
            case 128:
            case 129:
              {
                this.state = 1095;
                this.assemblyExpression();
              }
              break;
            default:
              throw new A(this);
          }
          this.state = 1098;
          this.assemblyBlock();
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    assemblyIf() {
      let localctx = new AssemblyIfContext(this, this._ctx, this.state);
      this.enterRule(localctx, 186, _SolidityParser.RULE_assemblyIf);
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 1100;
          this.match(_SolidityParser.T__50);
          this.state = 1101;
          this.assemblyExpression();
          this.state = 1102;
          this.assemblyBlock();
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    assemblyLiteral() {
      let localctx = new AssemblyLiteralContext(this, this._ctx, this.state);
      this.enterRule(localctx, 188, _SolidityParser.RULE_assemblyLiteral);
      try {
        this.state = 1109;
        this._errHandler.sync(this);
        switch (this._input.LA(1)) {
          case 129:
            this.enterOuterAlt(localctx, 1);
            {
              this.state = 1104;
              this.stringLiteral();
            }
            break;
          case 103:
            this.enterOuterAlt(localctx, 2);
            {
              this.state = 1105;
              this.match(_SolidityParser.DecimalNumber);
            }
            break;
          case 104:
            this.enterOuterAlt(localctx, 3);
            {
              this.state = 1106;
              this.match(_SolidityParser.HexNumber);
            }
            break;
          case 106:
            this.enterOuterAlt(localctx, 4);
            {
              this.state = 1107;
              this.hexLiteral();
            }
            break;
          case 102:
            this.enterOuterAlt(localctx, 5);
            {
              this.state = 1108;
              this.match(_SolidityParser.BooleanLiteral);
            }
            break;
          default:
            throw new A(this);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    tupleExpression() {
      let localctx = new TupleExpressionContext(this, this._ctx, this.state);
      this.enterRule(localctx, 190, _SolidityParser.RULE_tupleExpression);
      let _la;
      try {
        this.state = 1137;
        this._errHandler.sync(this);
        switch (this._input.LA(1)) {
          case 23:
            this.enterOuterAlt(localctx, 1);
            {
              this.state = 1111;
              this.match(_SolidityParser.T__22);
              {
                this.state = 1113;
                this._errHandler.sync(this);
                _la = this._input.LA(1);
                if ((_la & ~31) === 0 && (1 << _la & 3263184960) !== 0 || (_la - 38 & ~31) === 0 && (1 << _la - 38 & 4278194513) !== 0 || (_la - 71 & ~31) === 0 && (1 << _la - 71 & 4244635651) !== 0 || (_la - 103 & ~31) === 0 && (1 << _la - 103 & 124273675) !== 0) {
                  {
                    this.state = 1112;
                    this.expression(0);
                  }
                }
                this.state = 1121;
                this._errHandler.sync(this);
                _la = this._input.LA(1);
                while (_la === 16) {
                  {
                    {
                      this.state = 1115;
                      this.match(_SolidityParser.T__15);
                      this.state = 1117;
                      this._errHandler.sync(this);
                      _la = this._input.LA(1);
                      if ((_la & ~31) === 0 && (1 << _la & 3263184960) !== 0 || (_la - 38 & ~31) === 0 && (1 << _la - 38 & 4278194513) !== 0 || (_la - 71 & ~31) === 0 && (1 << _la - 71 & 4244635651) !== 0 || (_la - 103 & ~31) === 0 && (1 << _la - 103 & 124273675) !== 0) {
                        {
                          this.state = 1116;
                          this.expression(0);
                        }
                      }
                    }
                  }
                  this.state = 1123;
                  this._errHandler.sync(this);
                  _la = this._input.LA(1);
                }
              }
              this.state = 1124;
              this.match(_SolidityParser.T__23);
            }
            break;
          case 42:
            this.enterOuterAlt(localctx, 2);
            {
              this.state = 1125;
              this.match(_SolidityParser.T__41);
              this.state = 1134;
              this._errHandler.sync(this);
              _la = this._input.LA(1);
              if ((_la & ~31) === 0 && (1 << _la & 3263184960) !== 0 || (_la - 38 & ~31) === 0 && (1 << _la - 38 & 4278194513) !== 0 || (_la - 71 & ~31) === 0 && (1 << _la - 71 & 4244635651) !== 0 || (_la - 103 & ~31) === 0 && (1 << _la - 103 & 124273675) !== 0) {
                {
                  this.state = 1126;
                  this.expression(0);
                  this.state = 1131;
                  this._errHandler.sync(this);
                  _la = this._input.LA(1);
                  while (_la === 16) {
                    {
                      {
                        this.state = 1127;
                        this.match(_SolidityParser.T__15);
                        this.state = 1128;
                        this.expression(0);
                      }
                    }
                    this.state = 1133;
                    this._errHandler.sync(this);
                    _la = this._input.LA(1);
                  }
                }
              }
              this.state = 1136;
              this.match(_SolidityParser.T__42);
            }
            break;
          default:
            throw new A(this);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    numberLiteral() {
      let localctx = new NumberLiteralContext(this, this._ctx, this.state);
      this.enterRule(localctx, 192, _SolidityParser.RULE_numberLiteral);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 1139;
          _la = this._input.LA(1);
          if (!(_la === 103 || _la === 104)) {
            this._errHandler.recoverInline(this);
          } else {
            this._errHandler.reportMatch(this);
            this.consume();
          }
          this.state = 1141;
          this._errHandler.sync(this);
          switch (this._interp.adaptivePredict(this._input, 125, this._ctx)) {
            case 1:
              {
                this.state = 1140;
                this.match(_SolidityParser.NumberUnit);
              }
              break;
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    identifier() {
      let localctx = new IdentifierContext(this, this._ctx, this.state);
      this.enterRule(localctx, 194, _SolidityParser.RULE_identifier);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 1143;
          _la = this._input.LA(1);
          if (!(_la === 14 || _la === 25 || (_la - 44 & ~31) === 0 && (1 << _la - 44 & 262209) !== 0 || (_la - 95 & ~31) === 0 && (1 << _la - 95 & 1615069185) !== 0 || _la === 127 || _la === 128)) {
            this._errHandler.recoverInline(this);
          } else {
            this._errHandler.reportMatch(this);
            this.consume();
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    hexLiteral() {
      let localctx = new HexLiteralContext(this, this._ctx, this.state);
      this.enterRule(localctx, 196, _SolidityParser.RULE_hexLiteral);
      try {
        let _alt;
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 1146;
          this._errHandler.sync(this);
          _alt = 1;
          do {
            switch (_alt) {
              case 1:
                {
                  {
                    this.state = 1145;
                    this.match(_SolidityParser.HexLiteralFragment);
                  }
                }
                break;
              default:
                throw new A(this);
            }
            this.state = 1148;
            this._errHandler.sync(this);
            _alt = this._interp.adaptivePredict(this._input, 126, this._ctx);
          } while (_alt !== 2 && _alt !== i.INVALID_ALT_NUMBER);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    overrideSpecifier() {
      let localctx = new OverrideSpecifierContext(this, this._ctx, this.state);
      this.enterRule(localctx, 198, _SolidityParser.RULE_overrideSpecifier);
      let _la;
      try {
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 1150;
          this.match(_SolidityParser.T__95);
          this.state = 1162;
          this._errHandler.sync(this);
          _la = this._input.LA(1);
          if (_la === 23) {
            {
              this.state = 1151;
              this.match(_SolidityParser.T__22);
              this.state = 1152;
              this.userDefinedTypeName();
              this.state = 1157;
              this._errHandler.sync(this);
              _la = this._input.LA(1);
              while (_la === 16) {
                {
                  {
                    this.state = 1153;
                    this.match(_SolidityParser.T__15);
                    this.state = 1154;
                    this.userDefinedTypeName();
                  }
                }
                this.state = 1159;
                this._errHandler.sync(this);
                _la = this._input.LA(1);
              }
              this.state = 1160;
              this.match(_SolidityParser.T__23);
            }
          }
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    stringLiteral() {
      let localctx = new StringLiteralContext(this, this._ctx, this.state);
      this.enterRule(localctx, 200, _SolidityParser.RULE_stringLiteral);
      try {
        let _alt;
        this.enterOuterAlt(localctx, 1);
        {
          this.state = 1165;
          this._errHandler.sync(this);
          _alt = 1;
          do {
            switch (_alt) {
              case 1:
                {
                  {
                    this.state = 1164;
                    this.match(_SolidityParser.StringLiteralFragment);
                  }
                }
                break;
              default:
                throw new A(this);
            }
            this.state = 1167;
            this._errHandler.sync(this);
            _alt = this._interp.adaptivePredict(this._input, 129, this._ctx);
          } while (_alt !== 2 && _alt !== i.INVALID_ALT_NUMBER);
        }
      } catch (re) {
        if (re instanceof R) {
          localctx.exception = re;
          this._errHandler.reportError(this, re);
          this._errHandler.recover(this, re);
        } else {
          throw re;
        }
      } finally {
        this.exitRule();
      }
      return localctx;
    }
    sempred(localctx, ruleIndex, predIndex) {
      switch (ruleIndex) {
        case 38:
          return this.typeName_sempred(localctx, predIndex);
        case 70:
          return this.expression_sempred(localctx, predIndex);
      }
      return true;
    }
    typeName_sempred(localctx, predIndex) {
      switch (predIndex) {
        case 0:
          return this.precpred(this._ctx, 3);
      }
      return true;
    }
    expression_sempred(localctx, predIndex) {
      switch (predIndex) {
        case 1:
          return this.precpred(this._ctx, 14);
        case 2:
          return this.precpred(this._ctx, 13);
        case 3:
          return this.precpred(this._ctx, 12);
        case 4:
          return this.precpred(this._ctx, 11);
        case 5:
          return this.precpred(this._ctx, 10);
        case 6:
          return this.precpred(this._ctx, 9);
        case 7:
          return this.precpred(this._ctx, 8);
        case 8:
          return this.precpred(this._ctx, 7);
        case 9:
          return this.precpred(this._ctx, 6);
        case 10:
          return this.precpred(this._ctx, 5);
        case 11:
          return this.precpred(this._ctx, 4);
        case 12:
          return this.precpred(this._ctx, 3);
        case 13:
          return this.precpred(this._ctx, 2);
        case 14:
          return this.precpred(this._ctx, 27);
        case 15:
          return this.precpred(this._ctx, 25);
        case 16:
          return this.precpred(this._ctx, 24);
        case 17:
          return this.precpred(this._ctx, 23);
        case 18:
          return this.precpred(this._ctx, 22);
        case 19:
          return this.precpred(this._ctx, 21);
      }
      return true;
    }
    static get _ATN() {
      if (!_SolidityParser.__ATN) {
        _SolidityParser.__ATN = new r().deserialize(_SolidityParser._serializedATN);
      }
      return _SolidityParser.__ATN;
    }
  };
  var SolidityParser = _SolidityParser;
  SolidityParser.T__0 = 1;
  SolidityParser.T__1 = 2;
  SolidityParser.T__2 = 3;
  SolidityParser.T__3 = 4;
  SolidityParser.T__4 = 5;
  SolidityParser.T__5 = 6;
  SolidityParser.T__6 = 7;
  SolidityParser.T__7 = 8;
  SolidityParser.T__8 = 9;
  SolidityParser.T__9 = 10;
  SolidityParser.T__10 = 11;
  SolidityParser.T__11 = 12;
  SolidityParser.T__12 = 13;
  SolidityParser.T__13 = 14;
  SolidityParser.T__14 = 15;
  SolidityParser.T__15 = 16;
  SolidityParser.T__16 = 17;
  SolidityParser.T__17 = 18;
  SolidityParser.T__18 = 19;
  SolidityParser.T__19 = 20;
  SolidityParser.T__20 = 21;
  SolidityParser.T__21 = 22;
  SolidityParser.T__22 = 23;
  SolidityParser.T__23 = 24;
  SolidityParser.T__24 = 25;
  SolidityParser.T__25 = 26;
  SolidityParser.T__26 = 27;
  SolidityParser.T__27 = 28;
  SolidityParser.T__28 = 29;
  SolidityParser.T__29 = 30;
  SolidityParser.T__30 = 31;
  SolidityParser.T__31 = 32;
  SolidityParser.T__32 = 33;
  SolidityParser.T__33 = 34;
  SolidityParser.T__34 = 35;
  SolidityParser.T__35 = 36;
  SolidityParser.T__36 = 37;
  SolidityParser.T__37 = 38;
  SolidityParser.T__38 = 39;
  SolidityParser.T__39 = 40;
  SolidityParser.T__40 = 41;
  SolidityParser.T__41 = 42;
  SolidityParser.T__42 = 43;
  SolidityParser.T__43 = 44;
  SolidityParser.T__44 = 45;
  SolidityParser.T__45 = 46;
  SolidityParser.T__46 = 47;
  SolidityParser.T__47 = 48;
  SolidityParser.T__48 = 49;
  SolidityParser.T__49 = 50;
  SolidityParser.T__50 = 51;
  SolidityParser.T__51 = 52;
  SolidityParser.T__52 = 53;
  SolidityParser.T__53 = 54;
  SolidityParser.T__54 = 55;
  SolidityParser.T__55 = 56;
  SolidityParser.T__56 = 57;
  SolidityParser.T__57 = 58;
  SolidityParser.T__58 = 59;
  SolidityParser.T__59 = 60;
  SolidityParser.T__60 = 61;
  SolidityParser.T__61 = 62;
  SolidityParser.T__62 = 63;
  SolidityParser.T__63 = 64;
  SolidityParser.T__64 = 65;
  SolidityParser.T__65 = 66;
  SolidityParser.T__66 = 67;
  SolidityParser.T__67 = 68;
  SolidityParser.T__68 = 69;
  SolidityParser.T__69 = 70;
  SolidityParser.T__70 = 71;
  SolidityParser.T__71 = 72;
  SolidityParser.T__72 = 73;
  SolidityParser.T__73 = 74;
  SolidityParser.T__74 = 75;
  SolidityParser.T__75 = 76;
  SolidityParser.T__76 = 77;
  SolidityParser.T__77 = 78;
  SolidityParser.T__78 = 79;
  SolidityParser.T__79 = 80;
  SolidityParser.T__80 = 81;
  SolidityParser.T__81 = 82;
  SolidityParser.T__82 = 83;
  SolidityParser.T__83 = 84;
  SolidityParser.T__84 = 85;
  SolidityParser.T__85 = 86;
  SolidityParser.T__86 = 87;
  SolidityParser.T__87 = 88;
  SolidityParser.T__88 = 89;
  SolidityParser.T__89 = 90;
  SolidityParser.T__90 = 91;
  SolidityParser.T__91 = 92;
  SolidityParser.T__92 = 93;
  SolidityParser.T__93 = 94;
  SolidityParser.T__94 = 95;
  SolidityParser.T__95 = 96;
  SolidityParser.Int = 97;
  SolidityParser.Uint = 98;
  SolidityParser.Byte = 99;
  SolidityParser.Fixed = 100;
  SolidityParser.Ufixed = 101;
  SolidityParser.BooleanLiteral = 102;
  SolidityParser.DecimalNumber = 103;
  SolidityParser.HexNumber = 104;
  SolidityParser.NumberUnit = 105;
  SolidityParser.HexLiteralFragment = 106;
  SolidityParser.ReservedKeyword = 107;
  SolidityParser.AnonymousKeyword = 108;
  SolidityParser.BreakKeyword = 109;
  SolidityParser.ConstantKeyword = 110;
  SolidityParser.ImmutableKeyword = 111;
  SolidityParser.ContinueKeyword = 112;
  SolidityParser.LeaveKeyword = 113;
  SolidityParser.ExternalKeyword = 114;
  SolidityParser.IndexedKeyword = 115;
  SolidityParser.InternalKeyword = 116;
  SolidityParser.PayableKeyword = 117;
  SolidityParser.PrivateKeyword = 118;
  SolidityParser.PublicKeyword = 119;
  SolidityParser.VirtualKeyword = 120;
  SolidityParser.PureKeyword = 121;
  SolidityParser.TypeKeyword = 122;
  SolidityParser.ViewKeyword = 123;
  SolidityParser.GlobalKeyword = 124;
  SolidityParser.ConstructorKeyword = 125;
  SolidityParser.FallbackKeyword = 126;
  SolidityParser.ReceiveKeyword = 127;
  SolidityParser.Identifier = 128;
  SolidityParser.StringLiteralFragment = 129;
  SolidityParser.VersionLiteral = 130;
  SolidityParser.WS = 131;
  SolidityParser.COMMENT = 132;
  SolidityParser.LINE_COMMENT = 133;
  SolidityParser.EOF = D.EOF;
  SolidityParser.RULE_sourceUnit = 0;
  SolidityParser.RULE_pragmaDirective = 1;
  SolidityParser.RULE_pragmaName = 2;
  SolidityParser.RULE_pragmaValue = 3;
  SolidityParser.RULE_version = 4;
  SolidityParser.RULE_versionOperator = 5;
  SolidityParser.RULE_versionConstraint = 6;
  SolidityParser.RULE_importDeclaration = 7;
  SolidityParser.RULE_importDirective = 8;
  SolidityParser.RULE_importPath = 9;
  SolidityParser.RULE_contractDefinition = 10;
  SolidityParser.RULE_inheritanceSpecifier = 11;
  SolidityParser.RULE_contractPart = 12;
  SolidityParser.RULE_stateVariableDeclaration = 13;
  SolidityParser.RULE_fileLevelConstant = 14;
  SolidityParser.RULE_customErrorDefinition = 15;
  SolidityParser.RULE_typeDefinition = 16;
  SolidityParser.RULE_usingForDeclaration = 17;
  SolidityParser.RULE_usingForObject = 18;
  SolidityParser.RULE_usingForObjectDirective = 19;
  SolidityParser.RULE_userDefinableOperators = 20;
  SolidityParser.RULE_structDefinition = 21;
  SolidityParser.RULE_modifierDefinition = 22;
  SolidityParser.RULE_modifierInvocation = 23;
  SolidityParser.RULE_functionDefinition = 24;
  SolidityParser.RULE_functionDescriptor = 25;
  SolidityParser.RULE_returnParameters = 26;
  SolidityParser.RULE_modifierList = 27;
  SolidityParser.RULE_eventDefinition = 28;
  SolidityParser.RULE_enumValue = 29;
  SolidityParser.RULE_enumDefinition = 30;
  SolidityParser.RULE_parameterList = 31;
  SolidityParser.RULE_parameter = 32;
  SolidityParser.RULE_eventParameterList = 33;
  SolidityParser.RULE_eventParameter = 34;
  SolidityParser.RULE_functionTypeParameterList = 35;
  SolidityParser.RULE_functionTypeParameter = 36;
  SolidityParser.RULE_variableDeclaration = 37;
  SolidityParser.RULE_typeName = 38;
  SolidityParser.RULE_userDefinedTypeName = 39;
  SolidityParser.RULE_mappingKey = 40;
  SolidityParser.RULE_mapping = 41;
  SolidityParser.RULE_mappingKeyName = 42;
  SolidityParser.RULE_mappingValueName = 43;
  SolidityParser.RULE_functionTypeName = 44;
  SolidityParser.RULE_storageLocation = 45;
  SolidityParser.RULE_stateMutability = 46;
  SolidityParser.RULE_block = 47;
  SolidityParser.RULE_statement = 48;
  SolidityParser.RULE_expressionStatement = 49;
  SolidityParser.RULE_ifStatement = 50;
  SolidityParser.RULE_tryStatement = 51;
  SolidityParser.RULE_catchClause = 52;
  SolidityParser.RULE_whileStatement = 53;
  SolidityParser.RULE_simpleStatement = 54;
  SolidityParser.RULE_uncheckedStatement = 55;
  SolidityParser.RULE_forStatement = 56;
  SolidityParser.RULE_inlineAssemblyStatement = 57;
  SolidityParser.RULE_inlineAssemblyStatementFlag = 58;
  SolidityParser.RULE_doWhileStatement = 59;
  SolidityParser.RULE_continueStatement = 60;
  SolidityParser.RULE_breakStatement = 61;
  SolidityParser.RULE_returnStatement = 62;
  SolidityParser.RULE_throwStatement = 63;
  SolidityParser.RULE_emitStatement = 64;
  SolidityParser.RULE_revertStatement = 65;
  SolidityParser.RULE_variableDeclarationStatement = 66;
  SolidityParser.RULE_variableDeclarationList = 67;
  SolidityParser.RULE_identifierList = 68;
  SolidityParser.RULE_elementaryTypeName = 69;
  SolidityParser.RULE_expression = 70;
  SolidityParser.RULE_primaryExpression = 71;
  SolidityParser.RULE_expressionList = 72;
  SolidityParser.RULE_nameValueList = 73;
  SolidityParser.RULE_nameValue = 74;
  SolidityParser.RULE_functionCallArguments = 75;
  SolidityParser.RULE_functionCall = 76;
  SolidityParser.RULE_assemblyBlock = 77;
  SolidityParser.RULE_assemblyItem = 78;
  SolidityParser.RULE_assemblyExpression = 79;
  SolidityParser.RULE_assemblyMember = 80;
  SolidityParser.RULE_assemblyCall = 81;
  SolidityParser.RULE_assemblyLocalDefinition = 82;
  SolidityParser.RULE_assemblyAssignment = 83;
  SolidityParser.RULE_assemblyIdentifierOrList = 84;
  SolidityParser.RULE_assemblyIdentifierList = 85;
  SolidityParser.RULE_assemblyStackAssignment = 86;
  SolidityParser.RULE_labelDefinition = 87;
  SolidityParser.RULE_assemblySwitch = 88;
  SolidityParser.RULE_assemblyCase = 89;
  SolidityParser.RULE_assemblyFunctionDefinition = 90;
  SolidityParser.RULE_assemblyFunctionReturns = 91;
  SolidityParser.RULE_assemblyFor = 92;
  SolidityParser.RULE_assemblyIf = 93;
  SolidityParser.RULE_assemblyLiteral = 94;
  SolidityParser.RULE_tupleExpression = 95;
  SolidityParser.RULE_numberLiteral = 96;
  SolidityParser.RULE_identifier = 97;
  SolidityParser.RULE_hexLiteral = 98;
  SolidityParser.RULE_overrideSpecifier = 99;
  SolidityParser.RULE_stringLiteral = 100;
  SolidityParser.literalNames = [
    null,
    "'pragma'",
    "';'",
    "'*'",
    "'||'",
    "'^'",
    "'~'",
    "'>='",
    "'>'",
    "'<'",
    "'<='",
    "'='",
    "'as'",
    "'import'",
    "'from'",
    "'{'",
    "','",
    "'}'",
    "'abstract'",
    "'contract'",
    "'interface'",
    "'library'",
    "'is'",
    "'('",
    "')'",
    "'error'",
    "'using'",
    "'for'",
    "'|'",
    "'&'",
    "'+'",
    "'-'",
    "'/'",
    "'%'",
    "'=='",
    "'!='",
    "'struct'",
    "'modifier'",
    "'function'",
    "'returns'",
    "'event'",
    "'enum'",
    "'['",
    "']'",
    "'address'",
    "'.'",
    "'mapping'",
    "'=>'",
    "'memory'",
    "'storage'",
    "'calldata'",
    "'if'",
    "'else'",
    "'try'",
    "'catch'",
    "'while'",
    "'unchecked'",
    "'assembly'",
    "'do'",
    "'return'",
    "'throw'",
    "'emit'",
    "'revert'",
    "'var'",
    "'bool'",
    "'string'",
    "'byte'",
    "'++'",
    "'--'",
    "'new'",
    "':'",
    "'delete'",
    "'!'",
    "'**'",
    "'<<'",
    "'>>'",
    "'&&'",
    "'?'",
    "'|='",
    "'^='",
    "'&='",
    "'<<='",
    "'>>='",
    "'+='",
    "'-='",
    "'*='",
    "'/='",
    "'%='",
    "'let'",
    "':='",
    "'=:'",
    "'switch'",
    "'case'",
    "'default'",
    "'->'",
    "'callback'",
    "'override'",
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    "'anonymous'",
    "'break'",
    "'constant'",
    "'immutable'",
    "'continue'",
    "'leave'",
    "'external'",
    "'indexed'",
    "'internal'",
    "'payable'",
    "'private'",
    "'public'",
    "'virtual'",
    "'pure'",
    "'type'",
    "'view'",
    "'global'",
    "'constructor'",
    "'fallback'",
    "'receive'"
  ];
  SolidityParser.symbolicNames = [
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    "Int",
    "Uint",
    "Byte",
    "Fixed",
    "Ufixed",
    "BooleanLiteral",
    "DecimalNumber",
    "HexNumber",
    "NumberUnit",
    "HexLiteralFragment",
    "ReservedKeyword",
    "AnonymousKeyword",
    "BreakKeyword",
    "ConstantKeyword",
    "ImmutableKeyword",
    "ContinueKeyword",
    "LeaveKeyword",
    "ExternalKeyword",
    "IndexedKeyword",
    "InternalKeyword",
    "PayableKeyword",
    "PrivateKeyword",
    "PublicKeyword",
    "VirtualKeyword",
    "PureKeyword",
    "TypeKeyword",
    "ViewKeyword",
    "GlobalKeyword",
    "ConstructorKeyword",
    "FallbackKeyword",
    "ReceiveKeyword",
    "Identifier",
    "StringLiteralFragment",
    "VersionLiteral",
    "WS",
    "COMMENT",
    "LINE_COMMENT"
  ];
  SolidityParser.ruleNames = [
    "sourceUnit",
    "pragmaDirective",
    "pragmaName",
    "pragmaValue",
    "version",
    "versionOperator",
    "versionConstraint",
    "importDeclaration",
    "importDirective",
    "importPath",
    "contractDefinition",
    "inheritanceSpecifier",
    "contractPart",
    "stateVariableDeclaration",
    "fileLevelConstant",
    "customErrorDefinition",
    "typeDefinition",
    "usingForDeclaration",
    "usingForObject",
    "usingForObjectDirective",
    "userDefinableOperators",
    "structDefinition",
    "modifierDefinition",
    "modifierInvocation",
    "functionDefinition",
    "functionDescriptor",
    "returnParameters",
    "modifierList",
    "eventDefinition",
    "enumValue",
    "enumDefinition",
    "parameterList",
    "parameter",
    "eventParameterList",
    "eventParameter",
    "functionTypeParameterList",
    "functionTypeParameter",
    "variableDeclaration",
    "typeName",
    "userDefinedTypeName",
    "mappingKey",
    "mapping",
    "mappingKeyName",
    "mappingValueName",
    "functionTypeName",
    "storageLocation",
    "stateMutability",
    "block",
    "statement",
    "expressionStatement",
    "ifStatement",
    "tryStatement",
    "catchClause",
    "whileStatement",
    "simpleStatement",
    "uncheckedStatement",
    "forStatement",
    "inlineAssemblyStatement",
    "inlineAssemblyStatementFlag",
    "doWhileStatement",
    "continueStatement",
    "breakStatement",
    "returnStatement",
    "throwStatement",
    "emitStatement",
    "revertStatement",
    "variableDeclarationStatement",
    "variableDeclarationList",
    "identifierList",
    "elementaryTypeName",
    "expression",
    "primaryExpression",
    "expressionList",
    "nameValueList",
    "nameValue",
    "functionCallArguments",
    "functionCall",
    "assemblyBlock",
    "assemblyItem",
    "assemblyExpression",
    "assemblyMember",
    "assemblyCall",
    "assemblyLocalDefinition",
    "assemblyAssignment",
    "assemblyIdentifierOrList",
    "assemblyIdentifierList",
    "assemblyStackAssignment",
    "labelDefinition",
    "assemblySwitch",
    "assemblyCase",
    "assemblyFunctionDefinition",
    "assemblyFunctionReturns",
    "assemblyFor",
    "assemblyIf",
    "assemblyLiteral",
    "tupleExpression",
    "numberLiteral",
    "identifier",
    "hexLiteral",
    "overrideSpecifier",
    "stringLiteral"
  ];
  SolidityParser._serializedATN = [
    4,
    1,
    133,
    1170,
    2,
    0,
    7,
    0,
    2,
    1,
    7,
    1,
    2,
    2,
    7,
    2,
    2,
    3,
    7,
    3,
    2,
    4,
    7,
    4,
    2,
    5,
    7,
    5,
    2,
    6,
    7,
    6,
    2,
    7,
    7,
    7,
    2,
    8,
    7,
    8,
    2,
    9,
    7,
    9,
    2,
    10,
    7,
    10,
    2,
    11,
    7,
    11,
    2,
    12,
    7,
    12,
    2,
    13,
    7,
    13,
    2,
    14,
    7,
    14,
    2,
    15,
    7,
    15,
    2,
    16,
    7,
    16,
    2,
    17,
    7,
    17,
    2,
    18,
    7,
    18,
    2,
    19,
    7,
    19,
    2,
    20,
    7,
    20,
    2,
    21,
    7,
    21,
    2,
    22,
    7,
    22,
    2,
    23,
    7,
    23,
    2,
    24,
    7,
    24,
    2,
    25,
    7,
    25,
    2,
    26,
    7,
    26,
    2,
    27,
    7,
    27,
    2,
    28,
    7,
    28,
    2,
    29,
    7,
    29,
    2,
    30,
    7,
    30,
    2,
    31,
    7,
    31,
    2,
    32,
    7,
    32,
    2,
    33,
    7,
    33,
    2,
    34,
    7,
    34,
    2,
    35,
    7,
    35,
    2,
    36,
    7,
    36,
    2,
    37,
    7,
    37,
    2,
    38,
    7,
    38,
    2,
    39,
    7,
    39,
    2,
    40,
    7,
    40,
    2,
    41,
    7,
    41,
    2,
    42,
    7,
    42,
    2,
    43,
    7,
    43,
    2,
    44,
    7,
    44,
    2,
    45,
    7,
    45,
    2,
    46,
    7,
    46,
    2,
    47,
    7,
    47,
    2,
    48,
    7,
    48,
    2,
    49,
    7,
    49,
    2,
    50,
    7,
    50,
    2,
    51,
    7,
    51,
    2,
    52,
    7,
    52,
    2,
    53,
    7,
    53,
    2,
    54,
    7,
    54,
    2,
    55,
    7,
    55,
    2,
    56,
    7,
    56,
    2,
    57,
    7,
    57,
    2,
    58,
    7,
    58,
    2,
    59,
    7,
    59,
    2,
    60,
    7,
    60,
    2,
    61,
    7,
    61,
    2,
    62,
    7,
    62,
    2,
    63,
    7,
    63,
    2,
    64,
    7,
    64,
    2,
    65,
    7,
    65,
    2,
    66,
    7,
    66,
    2,
    67,
    7,
    67,
    2,
    68,
    7,
    68,
    2,
    69,
    7,
    69,
    2,
    70,
    7,
    70,
    2,
    71,
    7,
    71,
    2,
    72,
    7,
    72,
    2,
    73,
    7,
    73,
    2,
    74,
    7,
    74,
    2,
    75,
    7,
    75,
    2,
    76,
    7,
    76,
    2,
    77,
    7,
    77,
    2,
    78,
    7,
    78,
    2,
    79,
    7,
    79,
    2,
    80,
    7,
    80,
    2,
    81,
    7,
    81,
    2,
    82,
    7,
    82,
    2,
    83,
    7,
    83,
    2,
    84,
    7,
    84,
    2,
    85,
    7,
    85,
    2,
    86,
    7,
    86,
    2,
    87,
    7,
    87,
    2,
    88,
    7,
    88,
    2,
    89,
    7,
    89,
    2,
    90,
    7,
    90,
    2,
    91,
    7,
    91,
    2,
    92,
    7,
    92,
    2,
    93,
    7,
    93,
    2,
    94,
    7,
    94,
    2,
    95,
    7,
    95,
    2,
    96,
    7,
    96,
    2,
    97,
    7,
    97,
    2,
    98,
    7,
    98,
    2,
    99,
    7,
    99,
    2,
    100,
    7,
    100,
    1,
    0,
    1,
    0,
    1,
    0,
    1,
    0,
    1,
    0,
    1,
    0,
    1,
    0,
    1,
    0,
    1,
    0,
    1,
    0,
    1,
    0,
    5,
    0,
    214,
    8,
    0,
    10,
    0,
    12,
    0,
    217,
    9,
    0,
    1,
    0,
    1,
    0,
    1,
    1,
    1,
    1,
    1,
    1,
    1,
    1,
    1,
    1,
    1,
    2,
    1,
    2,
    1,
    3,
    1,
    3,
    1,
    3,
    3,
    3,
    231,
    8,
    3,
    1,
    4,
    1,
    4,
    3,
    4,
    235,
    8,
    4,
    1,
    4,
    5,
    4,
    238,
    8,
    4,
    10,
    4,
    12,
    4,
    241,
    9,
    4,
    1,
    5,
    1,
    5,
    1,
    6,
    3,
    6,
    246,
    8,
    6,
    1,
    6,
    1,
    6,
    3,
    6,
    250,
    8,
    6,
    1,
    6,
    3,
    6,
    253,
    8,
    6,
    1,
    7,
    1,
    7,
    1,
    7,
    3,
    7,
    258,
    8,
    7,
    1,
    8,
    1,
    8,
    1,
    8,
    1,
    8,
    3,
    8,
    264,
    8,
    8,
    1,
    8,
    1,
    8,
    1,
    8,
    1,
    8,
    1,
    8,
    3,
    8,
    271,
    8,
    8,
    1,
    8,
    1,
    8,
    3,
    8,
    275,
    8,
    8,
    1,
    8,
    1,
    8,
    1,
    8,
    1,
    8,
    1,
    8,
    1,
    8,
    1,
    8,
    1,
    8,
    1,
    8,
    5,
    8,
    286,
    8,
    8,
    10,
    8,
    12,
    8,
    289,
    9,
    8,
    1,
    8,
    1,
    8,
    1,
    8,
    1,
    8,
    1,
    8,
    3,
    8,
    296,
    8,
    8,
    1,
    9,
    1,
    9,
    1,
    10,
    3,
    10,
    301,
    8,
    10,
    1,
    10,
    1,
    10,
    1,
    10,
    1,
    10,
    1,
    10,
    1,
    10,
    5,
    10,
    309,
    8,
    10,
    10,
    10,
    12,
    10,
    312,
    9,
    10,
    3,
    10,
    314,
    8,
    10,
    1,
    10,
    1,
    10,
    5,
    10,
    318,
    8,
    10,
    10,
    10,
    12,
    10,
    321,
    9,
    10,
    1,
    10,
    1,
    10,
    1,
    11,
    1,
    11,
    1,
    11,
    3,
    11,
    328,
    8,
    11,
    1,
    11,
    3,
    11,
    331,
    8,
    11,
    1,
    12,
    1,
    12,
    1,
    12,
    1,
    12,
    1,
    12,
    1,
    12,
    1,
    12,
    1,
    12,
    1,
    12,
    3,
    12,
    342,
    8,
    12,
    1,
    13,
    1,
    13,
    1,
    13,
    1,
    13,
    1,
    13,
    1,
    13,
    1,
    13,
    5,
    13,
    351,
    8,
    13,
    10,
    13,
    12,
    13,
    354,
    9,
    13,
    1,
    13,
    1,
    13,
    1,
    13,
    3,
    13,
    359,
    8,
    13,
    1,
    13,
    1,
    13,
    1,
    14,
    1,
    14,
    1,
    14,
    1,
    14,
    1,
    14,
    1,
    14,
    1,
    14,
    1,
    15,
    1,
    15,
    1,
    15,
    1,
    15,
    1,
    15,
    1,
    16,
    1,
    16,
    1,
    16,
    1,
    16,
    1,
    16,
    1,
    16,
    1,
    17,
    1,
    17,
    1,
    17,
    1,
    17,
    1,
    17,
    3,
    17,
    386,
    8,
    17,
    1,
    17,
    3,
    17,
    389,
    8,
    17,
    1,
    17,
    1,
    17,
    1,
    18,
    1,
    18,
    1,
    18,
    1,
    18,
    1,
    18,
    5,
    18,
    398,
    8,
    18,
    10,
    18,
    12,
    18,
    401,
    9,
    18,
    1,
    18,
    1,
    18,
    3,
    18,
    405,
    8,
    18,
    1,
    19,
    1,
    19,
    1,
    19,
    3,
    19,
    410,
    8,
    19,
    1,
    20,
    1,
    20,
    1,
    21,
    1,
    21,
    1,
    21,
    1,
    21,
    1,
    21,
    1,
    21,
    1,
    21,
    1,
    21,
    5,
    21,
    422,
    8,
    21,
    10,
    21,
    12,
    21,
    425,
    9,
    21,
    3,
    21,
    427,
    8,
    21,
    1,
    21,
    1,
    21,
    1,
    22,
    1,
    22,
    1,
    22,
    3,
    22,
    434,
    8,
    22,
    1,
    22,
    1,
    22,
    5,
    22,
    438,
    8,
    22,
    10,
    22,
    12,
    22,
    441,
    9,
    22,
    1,
    22,
    1,
    22,
    3,
    22,
    445,
    8,
    22,
    1,
    23,
    1,
    23,
    1,
    23,
    3,
    23,
    450,
    8,
    23,
    1,
    23,
    3,
    23,
    453,
    8,
    23,
    1,
    24,
    1,
    24,
    1,
    24,
    1,
    24,
    3,
    24,
    459,
    8,
    24,
    1,
    24,
    1,
    24,
    3,
    24,
    463,
    8,
    24,
    1,
    25,
    1,
    25,
    3,
    25,
    467,
    8,
    25,
    1,
    25,
    1,
    25,
    1,
    25,
    3,
    25,
    472,
    8,
    25,
    1,
    26,
    1,
    26,
    1,
    26,
    1,
    27,
    1,
    27,
    1,
    27,
    1,
    27,
    1,
    27,
    1,
    27,
    1,
    27,
    1,
    27,
    5,
    27,
    485,
    8,
    27,
    10,
    27,
    12,
    27,
    488,
    9,
    27,
    1,
    28,
    1,
    28,
    1,
    28,
    1,
    28,
    3,
    28,
    494,
    8,
    28,
    1,
    28,
    1,
    28,
    1,
    29,
    1,
    29,
    1,
    30,
    1,
    30,
    1,
    30,
    1,
    30,
    3,
    30,
    504,
    8,
    30,
    1,
    30,
    1,
    30,
    5,
    30,
    508,
    8,
    30,
    10,
    30,
    12,
    30,
    511,
    9,
    30,
    1,
    30,
    1,
    30,
    1,
    31,
    1,
    31,
    1,
    31,
    1,
    31,
    5,
    31,
    519,
    8,
    31,
    10,
    31,
    12,
    31,
    522,
    9,
    31,
    3,
    31,
    524,
    8,
    31,
    1,
    31,
    1,
    31,
    1,
    32,
    1,
    32,
    3,
    32,
    530,
    8,
    32,
    1,
    32,
    3,
    32,
    533,
    8,
    32,
    1,
    33,
    1,
    33,
    1,
    33,
    1,
    33,
    5,
    33,
    539,
    8,
    33,
    10,
    33,
    12,
    33,
    542,
    9,
    33,
    3,
    33,
    544,
    8,
    33,
    1,
    33,
    1,
    33,
    1,
    34,
    1,
    34,
    3,
    34,
    550,
    8,
    34,
    1,
    34,
    3,
    34,
    553,
    8,
    34,
    1,
    35,
    1,
    35,
    1,
    35,
    1,
    35,
    5,
    35,
    559,
    8,
    35,
    10,
    35,
    12,
    35,
    562,
    9,
    35,
    3,
    35,
    564,
    8,
    35,
    1,
    35,
    1,
    35,
    1,
    36,
    1,
    36,
    3,
    36,
    570,
    8,
    36,
    1,
    37,
    1,
    37,
    3,
    37,
    574,
    8,
    37,
    1,
    37,
    1,
    37,
    1,
    38,
    1,
    38,
    1,
    38,
    1,
    38,
    1,
    38,
    1,
    38,
    1,
    38,
    3,
    38,
    585,
    8,
    38,
    1,
    38,
    1,
    38,
    1,
    38,
    3,
    38,
    590,
    8,
    38,
    1,
    38,
    5,
    38,
    593,
    8,
    38,
    10,
    38,
    12,
    38,
    596,
    9,
    38,
    1,
    39,
    1,
    39,
    1,
    39,
    5,
    39,
    601,
    8,
    39,
    10,
    39,
    12,
    39,
    604,
    9,
    39,
    1,
    40,
    1,
    40,
    3,
    40,
    608,
    8,
    40,
    1,
    41,
    1,
    41,
    1,
    41,
    1,
    41,
    3,
    41,
    614,
    8,
    41,
    1,
    41,
    1,
    41,
    1,
    41,
    3,
    41,
    619,
    8,
    41,
    1,
    41,
    1,
    41,
    1,
    42,
    1,
    42,
    1,
    43,
    1,
    43,
    1,
    44,
    1,
    44,
    1,
    44,
    1,
    44,
    1,
    44,
    5,
    44,
    632,
    8,
    44,
    10,
    44,
    12,
    44,
    635,
    9,
    44,
    1,
    44,
    1,
    44,
    3,
    44,
    639,
    8,
    44,
    1,
    45,
    1,
    45,
    1,
    46,
    1,
    46,
    1,
    47,
    1,
    47,
    5,
    47,
    647,
    8,
    47,
    10,
    47,
    12,
    47,
    650,
    9,
    47,
    1,
    47,
    1,
    47,
    1,
    48,
    1,
    48,
    1,
    48,
    1,
    48,
    1,
    48,
    1,
    48,
    1,
    48,
    1,
    48,
    1,
    48,
    1,
    48,
    1,
    48,
    1,
    48,
    1,
    48,
    1,
    48,
    1,
    48,
    3,
    48,
    669,
    8,
    48,
    1,
    49,
    1,
    49,
    1,
    49,
    1,
    50,
    1,
    50,
    1,
    50,
    1,
    50,
    1,
    50,
    1,
    50,
    1,
    50,
    3,
    50,
    681,
    8,
    50,
    1,
    51,
    1,
    51,
    1,
    51,
    3,
    51,
    686,
    8,
    51,
    1,
    51,
    1,
    51,
    4,
    51,
    690,
    8,
    51,
    11,
    51,
    12,
    51,
    691,
    1,
    52,
    1,
    52,
    3,
    52,
    696,
    8,
    52,
    1,
    52,
    3,
    52,
    699,
    8,
    52,
    1,
    52,
    1,
    52,
    1,
    53,
    1,
    53,
    1,
    53,
    1,
    53,
    1,
    53,
    1,
    53,
    1,
    54,
    1,
    54,
    3,
    54,
    711,
    8,
    54,
    1,
    55,
    1,
    55,
    1,
    55,
    1,
    56,
    1,
    56,
    1,
    56,
    1,
    56,
    3,
    56,
    720,
    8,
    56,
    1,
    56,
    1,
    56,
    3,
    56,
    724,
    8,
    56,
    1,
    56,
    3,
    56,
    727,
    8,
    56,
    1,
    56,
    1,
    56,
    1,
    56,
    1,
    57,
    1,
    57,
    3,
    57,
    734,
    8,
    57,
    1,
    57,
    1,
    57,
    1,
    57,
    1,
    57,
    3,
    57,
    740,
    8,
    57,
    1,
    57,
    1,
    57,
    1,
    58,
    1,
    58,
    1,
    59,
    1,
    59,
    1,
    59,
    1,
    59,
    1,
    59,
    1,
    59,
    1,
    59,
    1,
    59,
    1,
    60,
    1,
    60,
    1,
    60,
    1,
    61,
    1,
    61,
    1,
    61,
    1,
    62,
    1,
    62,
    3,
    62,
    762,
    8,
    62,
    1,
    62,
    1,
    62,
    1,
    63,
    1,
    63,
    1,
    63,
    1,
    64,
    1,
    64,
    1,
    64,
    1,
    64,
    1,
    65,
    1,
    65,
    1,
    65,
    1,
    65,
    1,
    66,
    1,
    66,
    1,
    66,
    1,
    66,
    1,
    66,
    1,
    66,
    1,
    66,
    3,
    66,
    784,
    8,
    66,
    1,
    66,
    1,
    66,
    3,
    66,
    788,
    8,
    66,
    1,
    66,
    1,
    66,
    1,
    67,
    3,
    67,
    793,
    8,
    67,
    1,
    67,
    1,
    67,
    3,
    67,
    797,
    8,
    67,
    5,
    67,
    799,
    8,
    67,
    10,
    67,
    12,
    67,
    802,
    9,
    67,
    1,
    68,
    1,
    68,
    3,
    68,
    806,
    8,
    68,
    1,
    68,
    5,
    68,
    809,
    8,
    68,
    10,
    68,
    12,
    68,
    812,
    9,
    68,
    1,
    68,
    3,
    68,
    815,
    8,
    68,
    1,
    68,
    1,
    68,
    1,
    69,
    1,
    69,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    3,
    70,
    839,
    8,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    3,
    70,
    893,
    8,
    70,
    1,
    70,
    1,
    70,
    3,
    70,
    897,
    8,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    1,
    70,
    5,
    70,
    913,
    8,
    70,
    10,
    70,
    12,
    70,
    916,
    9,
    70,
    1,
    71,
    1,
    71,
    1,
    71,
    1,
    71,
    1,
    71,
    1,
    71,
    1,
    71,
    1,
    71,
    1,
    71,
    3,
    71,
    927,
    8,
    71,
    1,
    72,
    1,
    72,
    1,
    72,
    5,
    72,
    932,
    8,
    72,
    10,
    72,
    12,
    72,
    935,
    9,
    72,
    1,
    73,
    1,
    73,
    1,
    73,
    5,
    73,
    940,
    8,
    73,
    10,
    73,
    12,
    73,
    943,
    9,
    73,
    1,
    73,
    3,
    73,
    946,
    8,
    73,
    1,
    74,
    1,
    74,
    1,
    74,
    1,
    74,
    1,
    75,
    1,
    75,
    3,
    75,
    954,
    8,
    75,
    1,
    75,
    1,
    75,
    3,
    75,
    958,
    8,
    75,
    3,
    75,
    960,
    8,
    75,
    1,
    76,
    1,
    76,
    1,
    76,
    1,
    76,
    1,
    76,
    1,
    77,
    1,
    77,
    5,
    77,
    969,
    8,
    77,
    10,
    77,
    12,
    77,
    972,
    9,
    77,
    1,
    77,
    1,
    77,
    1,
    78,
    1,
    78,
    1,
    78,
    1,
    78,
    1,
    78,
    1,
    78,
    1,
    78,
    1,
    78,
    1,
    78,
    1,
    78,
    1,
    78,
    1,
    78,
    1,
    78,
    1,
    78,
    1,
    78,
    1,
    78,
    1,
    78,
    3,
    78,
    993,
    8,
    78,
    1,
    79,
    1,
    79,
    1,
    79,
    3,
    79,
    998,
    8,
    79,
    1,
    80,
    1,
    80,
    1,
    80,
    1,
    80,
    1,
    81,
    1,
    81,
    1,
    81,
    1,
    81,
    3,
    81,
    1008,
    8,
    81,
    1,
    81,
    1,
    81,
    3,
    81,
    1012,
    8,
    81,
    1,
    81,
    1,
    81,
    5,
    81,
    1016,
    8,
    81,
    10,
    81,
    12,
    81,
    1019,
    9,
    81,
    1,
    81,
    3,
    81,
    1022,
    8,
    81,
    1,
    82,
    1,
    82,
    1,
    82,
    1,
    82,
    3,
    82,
    1028,
    8,
    82,
    1,
    83,
    1,
    83,
    1,
    83,
    1,
    83,
    1,
    84,
    1,
    84,
    1,
    84,
    1,
    84,
    1,
    84,
    1,
    84,
    1,
    84,
    3,
    84,
    1041,
    8,
    84,
    1,
    85,
    1,
    85,
    1,
    85,
    5,
    85,
    1046,
    8,
    85,
    10,
    85,
    12,
    85,
    1049,
    9,
    85,
    1,
    86,
    1,
    86,
    1,
    86,
    1,
    86,
    1,
    87,
    1,
    87,
    1,
    87,
    1,
    88,
    1,
    88,
    1,
    88,
    5,
    88,
    1061,
    8,
    88,
    10,
    88,
    12,
    88,
    1064,
    9,
    88,
    1,
    89,
    1,
    89,
    1,
    89,
    1,
    89,
    1,
    89,
    1,
    89,
    3,
    89,
    1072,
    8,
    89,
    1,
    90,
    1,
    90,
    1,
    90,
    1,
    90,
    3,
    90,
    1078,
    8,
    90,
    1,
    90,
    1,
    90,
    3,
    90,
    1082,
    8,
    90,
    1,
    90,
    1,
    90,
    1,
    91,
    1,
    91,
    1,
    91,
    1,
    92,
    1,
    92,
    1,
    92,
    3,
    92,
    1092,
    8,
    92,
    1,
    92,
    1,
    92,
    1,
    92,
    3,
    92,
    1097,
    8,
    92,
    1,
    92,
    1,
    92,
    1,
    93,
    1,
    93,
    1,
    93,
    1,
    93,
    1,
    94,
    1,
    94,
    1,
    94,
    1,
    94,
    1,
    94,
    3,
    94,
    1110,
    8,
    94,
    1,
    95,
    1,
    95,
    3,
    95,
    1114,
    8,
    95,
    1,
    95,
    1,
    95,
    3,
    95,
    1118,
    8,
    95,
    5,
    95,
    1120,
    8,
    95,
    10,
    95,
    12,
    95,
    1123,
    9,
    95,
    1,
    95,
    1,
    95,
    1,
    95,
    1,
    95,
    1,
    95,
    5,
    95,
    1130,
    8,
    95,
    10,
    95,
    12,
    95,
    1133,
    9,
    95,
    3,
    95,
    1135,
    8,
    95,
    1,
    95,
    3,
    95,
    1138,
    8,
    95,
    1,
    96,
    1,
    96,
    3,
    96,
    1142,
    8,
    96,
    1,
    97,
    1,
    97,
    1,
    98,
    4,
    98,
    1147,
    8,
    98,
    11,
    98,
    12,
    98,
    1148,
    1,
    99,
    1,
    99,
    1,
    99,
    1,
    99,
    1,
    99,
    5,
    99,
    1156,
    8,
    99,
    10,
    99,
    12,
    99,
    1159,
    9,
    99,
    1,
    99,
    1,
    99,
    3,
    99,
    1163,
    8,
    99,
    1,
    100,
    4,
    100,
    1166,
    8,
    100,
    11,
    100,
    12,
    100,
    1167,
    1,
    100,
    0,
    2,
    76,
    140,
    101,
    0,
    2,
    4,
    6,
    8,
    10,
    12,
    14,
    16,
    18,
    20,
    22,
    24,
    26,
    28,
    30,
    32,
    34,
    36,
    38,
    40,
    42,
    44,
    46,
    48,
    50,
    52,
    54,
    56,
    58,
    60,
    62,
    64,
    66,
    68,
    70,
    72,
    74,
    76,
    78,
    80,
    82,
    84,
    86,
    88,
    90,
    92,
    94,
    96,
    98,
    100,
    102,
    104,
    106,
    108,
    110,
    112,
    114,
    116,
    118,
    120,
    122,
    124,
    126,
    128,
    130,
    132,
    134,
    136,
    138,
    140,
    142,
    144,
    146,
    148,
    150,
    152,
    154,
    156,
    158,
    160,
    162,
    164,
    166,
    168,
    170,
    172,
    174,
    176,
    178,
    180,
    182,
    184,
    186,
    188,
    190,
    192,
    194,
    196,
    198,
    200,
    0,
    15,
    1,
    0,
    5,
    11,
    1,
    0,
    19,
    21,
    3,
    0,
    3,
    3,
    5,
    10,
    28,
    35,
    1,
    0,
    48,
    50,
    4,
    0,
    110,
    110,
    117,
    117,
    121,
    121,
    123,
    123,
    3,
    0,
    44,
    44,
    63,
    66,
    97,
    101,
    1,
    0,
    67,
    68,
    1,
    0,
    30,
    31,
    2,
    0,
    3,
    3,
    32,
    33,
    1,
    0,
    74,
    75,
    1,
    0,
    7,
    10,
    1,
    0,
    34,
    35,
    2,
    0,
    11,
    11,
    78,
    87,
    1,
    0,
    103,
    104,
    10,
    0,
    14,
    14,
    25,
    25,
    44,
    44,
    50,
    50,
    62,
    62,
    95,
    95,
    113,
    113,
    117,
    117,
    124,
    125,
    127,
    128,
    1299,
    0,
    215,
    1,
    0,
    0,
    0,
    2,
    220,
    1,
    0,
    0,
    0,
    4,
    225,
    1,
    0,
    0,
    0,
    6,
    230,
    1,
    0,
    0,
    0,
    8,
    232,
    1,
    0,
    0,
    0,
    10,
    242,
    1,
    0,
    0,
    0,
    12,
    252,
    1,
    0,
    0,
    0,
    14,
    254,
    1,
    0,
    0,
    0,
    16,
    295,
    1,
    0,
    0,
    0,
    18,
    297,
    1,
    0,
    0,
    0,
    20,
    300,
    1,
    0,
    0,
    0,
    22,
    324,
    1,
    0,
    0,
    0,
    24,
    341,
    1,
    0,
    0,
    0,
    26,
    343,
    1,
    0,
    0,
    0,
    28,
    362,
    1,
    0,
    0,
    0,
    30,
    369,
    1,
    0,
    0,
    0,
    32,
    374,
    1,
    0,
    0,
    0,
    34,
    380,
    1,
    0,
    0,
    0,
    36,
    404,
    1,
    0,
    0,
    0,
    38,
    406,
    1,
    0,
    0,
    0,
    40,
    411,
    1,
    0,
    0,
    0,
    42,
    413,
    1,
    0,
    0,
    0,
    44,
    430,
    1,
    0,
    0,
    0,
    46,
    446,
    1,
    0,
    0,
    0,
    48,
    454,
    1,
    0,
    0,
    0,
    50,
    471,
    1,
    0,
    0,
    0,
    52,
    473,
    1,
    0,
    0,
    0,
    54,
    486,
    1,
    0,
    0,
    0,
    56,
    489,
    1,
    0,
    0,
    0,
    58,
    497,
    1,
    0,
    0,
    0,
    60,
    499,
    1,
    0,
    0,
    0,
    62,
    514,
    1,
    0,
    0,
    0,
    64,
    527,
    1,
    0,
    0,
    0,
    66,
    534,
    1,
    0,
    0,
    0,
    68,
    547,
    1,
    0,
    0,
    0,
    70,
    554,
    1,
    0,
    0,
    0,
    72,
    567,
    1,
    0,
    0,
    0,
    74,
    571,
    1,
    0,
    0,
    0,
    76,
    584,
    1,
    0,
    0,
    0,
    78,
    597,
    1,
    0,
    0,
    0,
    80,
    607,
    1,
    0,
    0,
    0,
    82,
    609,
    1,
    0,
    0,
    0,
    84,
    622,
    1,
    0,
    0,
    0,
    86,
    624,
    1,
    0,
    0,
    0,
    88,
    626,
    1,
    0,
    0,
    0,
    90,
    640,
    1,
    0,
    0,
    0,
    92,
    642,
    1,
    0,
    0,
    0,
    94,
    644,
    1,
    0,
    0,
    0,
    96,
    668,
    1,
    0,
    0,
    0,
    98,
    670,
    1,
    0,
    0,
    0,
    100,
    673,
    1,
    0,
    0,
    0,
    102,
    682,
    1,
    0,
    0,
    0,
    104,
    693,
    1,
    0,
    0,
    0,
    106,
    702,
    1,
    0,
    0,
    0,
    108,
    710,
    1,
    0,
    0,
    0,
    110,
    712,
    1,
    0,
    0,
    0,
    112,
    715,
    1,
    0,
    0,
    0,
    114,
    731,
    1,
    0,
    0,
    0,
    116,
    743,
    1,
    0,
    0,
    0,
    118,
    745,
    1,
    0,
    0,
    0,
    120,
    753,
    1,
    0,
    0,
    0,
    122,
    756,
    1,
    0,
    0,
    0,
    124,
    759,
    1,
    0,
    0,
    0,
    126,
    765,
    1,
    0,
    0,
    0,
    128,
    768,
    1,
    0,
    0,
    0,
    130,
    772,
    1,
    0,
    0,
    0,
    132,
    783,
    1,
    0,
    0,
    0,
    134,
    792,
    1,
    0,
    0,
    0,
    136,
    803,
    1,
    0,
    0,
    0,
    138,
    818,
    1,
    0,
    0,
    0,
    140,
    838,
    1,
    0,
    0,
    0,
    142,
    926,
    1,
    0,
    0,
    0,
    144,
    928,
    1,
    0,
    0,
    0,
    146,
    936,
    1,
    0,
    0,
    0,
    148,
    947,
    1,
    0,
    0,
    0,
    150,
    959,
    1,
    0,
    0,
    0,
    152,
    961,
    1,
    0,
    0,
    0,
    154,
    966,
    1,
    0,
    0,
    0,
    156,
    992,
    1,
    0,
    0,
    0,
    158,
    997,
    1,
    0,
    0,
    0,
    160,
    999,
    1,
    0,
    0,
    0,
    162,
    1007,
    1,
    0,
    0,
    0,
    164,
    1023,
    1,
    0,
    0,
    0,
    166,
    1029,
    1,
    0,
    0,
    0,
    168,
    1040,
    1,
    0,
    0,
    0,
    170,
    1042,
    1,
    0,
    0,
    0,
    172,
    1050,
    1,
    0,
    0,
    0,
    174,
    1054,
    1,
    0,
    0,
    0,
    176,
    1057,
    1,
    0,
    0,
    0,
    178,
    1071,
    1,
    0,
    0,
    0,
    180,
    1073,
    1,
    0,
    0,
    0,
    182,
    1085,
    1,
    0,
    0,
    0,
    184,
    1088,
    1,
    0,
    0,
    0,
    186,
    1100,
    1,
    0,
    0,
    0,
    188,
    1109,
    1,
    0,
    0,
    0,
    190,
    1137,
    1,
    0,
    0,
    0,
    192,
    1139,
    1,
    0,
    0,
    0,
    194,
    1143,
    1,
    0,
    0,
    0,
    196,
    1146,
    1,
    0,
    0,
    0,
    198,
    1150,
    1,
    0,
    0,
    0,
    200,
    1165,
    1,
    0,
    0,
    0,
    202,
    214,
    3,
    2,
    1,
    0,
    203,
    214,
    3,
    16,
    8,
    0,
    204,
    214,
    3,
    20,
    10,
    0,
    205,
    214,
    3,
    60,
    30,
    0,
    206,
    214,
    3,
    56,
    28,
    0,
    207,
    214,
    3,
    42,
    21,
    0,
    208,
    214,
    3,
    48,
    24,
    0,
    209,
    214,
    3,
    28,
    14,
    0,
    210,
    214,
    3,
    30,
    15,
    0,
    211,
    214,
    3,
    32,
    16,
    0,
    212,
    214,
    3,
    34,
    17,
    0,
    213,
    202,
    1,
    0,
    0,
    0,
    213,
    203,
    1,
    0,
    0,
    0,
    213,
    204,
    1,
    0,
    0,
    0,
    213,
    205,
    1,
    0,
    0,
    0,
    213,
    206,
    1,
    0,
    0,
    0,
    213,
    207,
    1,
    0,
    0,
    0,
    213,
    208,
    1,
    0,
    0,
    0,
    213,
    209,
    1,
    0,
    0,
    0,
    213,
    210,
    1,
    0,
    0,
    0,
    213,
    211,
    1,
    0,
    0,
    0,
    213,
    212,
    1,
    0,
    0,
    0,
    214,
    217,
    1,
    0,
    0,
    0,
    215,
    213,
    1,
    0,
    0,
    0,
    215,
    216,
    1,
    0,
    0,
    0,
    216,
    218,
    1,
    0,
    0,
    0,
    217,
    215,
    1,
    0,
    0,
    0,
    218,
    219,
    5,
    0,
    0,
    1,
    219,
    1,
    1,
    0,
    0,
    0,
    220,
    221,
    5,
    1,
    0,
    0,
    221,
    222,
    3,
    4,
    2,
    0,
    222,
    223,
    3,
    6,
    3,
    0,
    223,
    224,
    5,
    2,
    0,
    0,
    224,
    3,
    1,
    0,
    0,
    0,
    225,
    226,
    3,
    194,
    97,
    0,
    226,
    5,
    1,
    0,
    0,
    0,
    227,
    231,
    5,
    3,
    0,
    0,
    228,
    231,
    3,
    8,
    4,
    0,
    229,
    231,
    3,
    140,
    70,
    0,
    230,
    227,
    1,
    0,
    0,
    0,
    230,
    228,
    1,
    0,
    0,
    0,
    230,
    229,
    1,
    0,
    0,
    0,
    231,
    7,
    1,
    0,
    0,
    0,
    232,
    239,
    3,
    12,
    6,
    0,
    233,
    235,
    5,
    4,
    0,
    0,
    234,
    233,
    1,
    0,
    0,
    0,
    234,
    235,
    1,
    0,
    0,
    0,
    235,
    236,
    1,
    0,
    0,
    0,
    236,
    238,
    3,
    12,
    6,
    0,
    237,
    234,
    1,
    0,
    0,
    0,
    238,
    241,
    1,
    0,
    0,
    0,
    239,
    237,
    1,
    0,
    0,
    0,
    239,
    240,
    1,
    0,
    0,
    0,
    240,
    9,
    1,
    0,
    0,
    0,
    241,
    239,
    1,
    0,
    0,
    0,
    242,
    243,
    7,
    0,
    0,
    0,
    243,
    11,
    1,
    0,
    0,
    0,
    244,
    246,
    3,
    10,
    5,
    0,
    245,
    244,
    1,
    0,
    0,
    0,
    245,
    246,
    1,
    0,
    0,
    0,
    246,
    247,
    1,
    0,
    0,
    0,
    247,
    253,
    5,
    130,
    0,
    0,
    248,
    250,
    3,
    10,
    5,
    0,
    249,
    248,
    1,
    0,
    0,
    0,
    249,
    250,
    1,
    0,
    0,
    0,
    250,
    251,
    1,
    0,
    0,
    0,
    251,
    253,
    5,
    103,
    0,
    0,
    252,
    245,
    1,
    0,
    0,
    0,
    252,
    249,
    1,
    0,
    0,
    0,
    253,
    13,
    1,
    0,
    0,
    0,
    254,
    257,
    3,
    194,
    97,
    0,
    255,
    256,
    5,
    12,
    0,
    0,
    256,
    258,
    3,
    194,
    97,
    0,
    257,
    255,
    1,
    0,
    0,
    0,
    257,
    258,
    1,
    0,
    0,
    0,
    258,
    15,
    1,
    0,
    0,
    0,
    259,
    260,
    5,
    13,
    0,
    0,
    260,
    263,
    3,
    18,
    9,
    0,
    261,
    262,
    5,
    12,
    0,
    0,
    262,
    264,
    3,
    194,
    97,
    0,
    263,
    261,
    1,
    0,
    0,
    0,
    263,
    264,
    1,
    0,
    0,
    0,
    264,
    265,
    1,
    0,
    0,
    0,
    265,
    266,
    5,
    2,
    0,
    0,
    266,
    296,
    1,
    0,
    0,
    0,
    267,
    270,
    5,
    13,
    0,
    0,
    268,
    271,
    5,
    3,
    0,
    0,
    269,
    271,
    3,
    194,
    97,
    0,
    270,
    268,
    1,
    0,
    0,
    0,
    270,
    269,
    1,
    0,
    0,
    0,
    271,
    274,
    1,
    0,
    0,
    0,
    272,
    273,
    5,
    12,
    0,
    0,
    273,
    275,
    3,
    194,
    97,
    0,
    274,
    272,
    1,
    0,
    0,
    0,
    274,
    275,
    1,
    0,
    0,
    0,
    275,
    276,
    1,
    0,
    0,
    0,
    276,
    277,
    5,
    14,
    0,
    0,
    277,
    278,
    3,
    18,
    9,
    0,
    278,
    279,
    5,
    2,
    0,
    0,
    279,
    296,
    1,
    0,
    0,
    0,
    280,
    281,
    5,
    13,
    0,
    0,
    281,
    282,
    5,
    15,
    0,
    0,
    282,
    287,
    3,
    14,
    7,
    0,
    283,
    284,
    5,
    16,
    0,
    0,
    284,
    286,
    3,
    14,
    7,
    0,
    285,
    283,
    1,
    0,
    0,
    0,
    286,
    289,
    1,
    0,
    0,
    0,
    287,
    285,
    1,
    0,
    0,
    0,
    287,
    288,
    1,
    0,
    0,
    0,
    288,
    290,
    1,
    0,
    0,
    0,
    289,
    287,
    1,
    0,
    0,
    0,
    290,
    291,
    5,
    17,
    0,
    0,
    291,
    292,
    5,
    14,
    0,
    0,
    292,
    293,
    3,
    18,
    9,
    0,
    293,
    294,
    5,
    2,
    0,
    0,
    294,
    296,
    1,
    0,
    0,
    0,
    295,
    259,
    1,
    0,
    0,
    0,
    295,
    267,
    1,
    0,
    0,
    0,
    295,
    280,
    1,
    0,
    0,
    0,
    296,
    17,
    1,
    0,
    0,
    0,
    297,
    298,
    5,
    129,
    0,
    0,
    298,
    19,
    1,
    0,
    0,
    0,
    299,
    301,
    5,
    18,
    0,
    0,
    300,
    299,
    1,
    0,
    0,
    0,
    300,
    301,
    1,
    0,
    0,
    0,
    301,
    302,
    1,
    0,
    0,
    0,
    302,
    303,
    7,
    1,
    0,
    0,
    303,
    313,
    3,
    194,
    97,
    0,
    304,
    305,
    5,
    22,
    0,
    0,
    305,
    310,
    3,
    22,
    11,
    0,
    306,
    307,
    5,
    16,
    0,
    0,
    307,
    309,
    3,
    22,
    11,
    0,
    308,
    306,
    1,
    0,
    0,
    0,
    309,
    312,
    1,
    0,
    0,
    0,
    310,
    308,
    1,
    0,
    0,
    0,
    310,
    311,
    1,
    0,
    0,
    0,
    311,
    314,
    1,
    0,
    0,
    0,
    312,
    310,
    1,
    0,
    0,
    0,
    313,
    304,
    1,
    0,
    0,
    0,
    313,
    314,
    1,
    0,
    0,
    0,
    314,
    315,
    1,
    0,
    0,
    0,
    315,
    319,
    5,
    15,
    0,
    0,
    316,
    318,
    3,
    24,
    12,
    0,
    317,
    316,
    1,
    0,
    0,
    0,
    318,
    321,
    1,
    0,
    0,
    0,
    319,
    317,
    1,
    0,
    0,
    0,
    319,
    320,
    1,
    0,
    0,
    0,
    320,
    322,
    1,
    0,
    0,
    0,
    321,
    319,
    1,
    0,
    0,
    0,
    322,
    323,
    5,
    17,
    0,
    0,
    323,
    21,
    1,
    0,
    0,
    0,
    324,
    330,
    3,
    78,
    39,
    0,
    325,
    327,
    5,
    23,
    0,
    0,
    326,
    328,
    3,
    144,
    72,
    0,
    327,
    326,
    1,
    0,
    0,
    0,
    327,
    328,
    1,
    0,
    0,
    0,
    328,
    329,
    1,
    0,
    0,
    0,
    329,
    331,
    5,
    24,
    0,
    0,
    330,
    325,
    1,
    0,
    0,
    0,
    330,
    331,
    1,
    0,
    0,
    0,
    331,
    23,
    1,
    0,
    0,
    0,
    332,
    342,
    3,
    26,
    13,
    0,
    333,
    342,
    3,
    34,
    17,
    0,
    334,
    342,
    3,
    42,
    21,
    0,
    335,
    342,
    3,
    44,
    22,
    0,
    336,
    342,
    3,
    48,
    24,
    0,
    337,
    342,
    3,
    56,
    28,
    0,
    338,
    342,
    3,
    60,
    30,
    0,
    339,
    342,
    3,
    30,
    15,
    0,
    340,
    342,
    3,
    32,
    16,
    0,
    341,
    332,
    1,
    0,
    0,
    0,
    341,
    333,
    1,
    0,
    0,
    0,
    341,
    334,
    1,
    0,
    0,
    0,
    341,
    335,
    1,
    0,
    0,
    0,
    341,
    336,
    1,
    0,
    0,
    0,
    341,
    337,
    1,
    0,
    0,
    0,
    341,
    338,
    1,
    0,
    0,
    0,
    341,
    339,
    1,
    0,
    0,
    0,
    341,
    340,
    1,
    0,
    0,
    0,
    342,
    25,
    1,
    0,
    0,
    0,
    343,
    352,
    3,
    76,
    38,
    0,
    344,
    351,
    5,
    119,
    0,
    0,
    345,
    351,
    5,
    116,
    0,
    0,
    346,
    351,
    5,
    118,
    0,
    0,
    347,
    351,
    5,
    110,
    0,
    0,
    348,
    351,
    5,
    111,
    0,
    0,
    349,
    351,
    3,
    198,
    99,
    0,
    350,
    344,
    1,
    0,
    0,
    0,
    350,
    345,
    1,
    0,
    0,
    0,
    350,
    346,
    1,
    0,
    0,
    0,
    350,
    347,
    1,
    0,
    0,
    0,
    350,
    348,
    1,
    0,
    0,
    0,
    350,
    349,
    1,
    0,
    0,
    0,
    351,
    354,
    1,
    0,
    0,
    0,
    352,
    350,
    1,
    0,
    0,
    0,
    352,
    353,
    1,
    0,
    0,
    0,
    353,
    355,
    1,
    0,
    0,
    0,
    354,
    352,
    1,
    0,
    0,
    0,
    355,
    358,
    3,
    194,
    97,
    0,
    356,
    357,
    5,
    11,
    0,
    0,
    357,
    359,
    3,
    140,
    70,
    0,
    358,
    356,
    1,
    0,
    0,
    0,
    358,
    359,
    1,
    0,
    0,
    0,
    359,
    360,
    1,
    0,
    0,
    0,
    360,
    361,
    5,
    2,
    0,
    0,
    361,
    27,
    1,
    0,
    0,
    0,
    362,
    363,
    3,
    76,
    38,
    0,
    363,
    364,
    5,
    110,
    0,
    0,
    364,
    365,
    3,
    194,
    97,
    0,
    365,
    366,
    5,
    11,
    0,
    0,
    366,
    367,
    3,
    140,
    70,
    0,
    367,
    368,
    5,
    2,
    0,
    0,
    368,
    29,
    1,
    0,
    0,
    0,
    369,
    370,
    5,
    25,
    0,
    0,
    370,
    371,
    3,
    194,
    97,
    0,
    371,
    372,
    3,
    62,
    31,
    0,
    372,
    373,
    5,
    2,
    0,
    0,
    373,
    31,
    1,
    0,
    0,
    0,
    374,
    375,
    5,
    122,
    0,
    0,
    375,
    376,
    3,
    194,
    97,
    0,
    376,
    377,
    5,
    22,
    0,
    0,
    377,
    378,
    3,
    138,
    69,
    0,
    378,
    379,
    5,
    2,
    0,
    0,
    379,
    33,
    1,
    0,
    0,
    0,
    380,
    381,
    5,
    26,
    0,
    0,
    381,
    382,
    3,
    36,
    18,
    0,
    382,
    385,
    5,
    27,
    0,
    0,
    383,
    386,
    5,
    3,
    0,
    0,
    384,
    386,
    3,
    76,
    38,
    0,
    385,
    383,
    1,
    0,
    0,
    0,
    385,
    384,
    1,
    0,
    0,
    0,
    386,
    388,
    1,
    0,
    0,
    0,
    387,
    389,
    5,
    124,
    0,
    0,
    388,
    387,
    1,
    0,
    0,
    0,
    388,
    389,
    1,
    0,
    0,
    0,
    389,
    390,
    1,
    0,
    0,
    0,
    390,
    391,
    5,
    2,
    0,
    0,
    391,
    35,
    1,
    0,
    0,
    0,
    392,
    405,
    3,
    78,
    39,
    0,
    393,
    394,
    5,
    15,
    0,
    0,
    394,
    399,
    3,
    38,
    19,
    0,
    395,
    396,
    5,
    16,
    0,
    0,
    396,
    398,
    3,
    38,
    19,
    0,
    397,
    395,
    1,
    0,
    0,
    0,
    398,
    401,
    1,
    0,
    0,
    0,
    399,
    397,
    1,
    0,
    0,
    0,
    399,
    400,
    1,
    0,
    0,
    0,
    400,
    402,
    1,
    0,
    0,
    0,
    401,
    399,
    1,
    0,
    0,
    0,
    402,
    403,
    5,
    17,
    0,
    0,
    403,
    405,
    1,
    0,
    0,
    0,
    404,
    392,
    1,
    0,
    0,
    0,
    404,
    393,
    1,
    0,
    0,
    0,
    405,
    37,
    1,
    0,
    0,
    0,
    406,
    409,
    3,
    78,
    39,
    0,
    407,
    408,
    5,
    12,
    0,
    0,
    408,
    410,
    3,
    40,
    20,
    0,
    409,
    407,
    1,
    0,
    0,
    0,
    409,
    410,
    1,
    0,
    0,
    0,
    410,
    39,
    1,
    0,
    0,
    0,
    411,
    412,
    7,
    2,
    0,
    0,
    412,
    41,
    1,
    0,
    0,
    0,
    413,
    414,
    5,
    36,
    0,
    0,
    414,
    415,
    3,
    194,
    97,
    0,
    415,
    426,
    5,
    15,
    0,
    0,
    416,
    417,
    3,
    74,
    37,
    0,
    417,
    423,
    5,
    2,
    0,
    0,
    418,
    419,
    3,
    74,
    37,
    0,
    419,
    420,
    5,
    2,
    0,
    0,
    420,
    422,
    1,
    0,
    0,
    0,
    421,
    418,
    1,
    0,
    0,
    0,
    422,
    425,
    1,
    0,
    0,
    0,
    423,
    421,
    1,
    0,
    0,
    0,
    423,
    424,
    1,
    0,
    0,
    0,
    424,
    427,
    1,
    0,
    0,
    0,
    425,
    423,
    1,
    0,
    0,
    0,
    426,
    416,
    1,
    0,
    0,
    0,
    426,
    427,
    1,
    0,
    0,
    0,
    427,
    428,
    1,
    0,
    0,
    0,
    428,
    429,
    5,
    17,
    0,
    0,
    429,
    43,
    1,
    0,
    0,
    0,
    430,
    431,
    5,
    37,
    0,
    0,
    431,
    433,
    3,
    194,
    97,
    0,
    432,
    434,
    3,
    62,
    31,
    0,
    433,
    432,
    1,
    0,
    0,
    0,
    433,
    434,
    1,
    0,
    0,
    0,
    434,
    439,
    1,
    0,
    0,
    0,
    435,
    438,
    5,
    120,
    0,
    0,
    436,
    438,
    3,
    198,
    99,
    0,
    437,
    435,
    1,
    0,
    0,
    0,
    437,
    436,
    1,
    0,
    0,
    0,
    438,
    441,
    1,
    0,
    0,
    0,
    439,
    437,
    1,
    0,
    0,
    0,
    439,
    440,
    1,
    0,
    0,
    0,
    440,
    444,
    1,
    0,
    0,
    0,
    441,
    439,
    1,
    0,
    0,
    0,
    442,
    445,
    5,
    2,
    0,
    0,
    443,
    445,
    3,
    94,
    47,
    0,
    444,
    442,
    1,
    0,
    0,
    0,
    444,
    443,
    1,
    0,
    0,
    0,
    445,
    45,
    1,
    0,
    0,
    0,
    446,
    452,
    3,
    194,
    97,
    0,
    447,
    449,
    5,
    23,
    0,
    0,
    448,
    450,
    3,
    144,
    72,
    0,
    449,
    448,
    1,
    0,
    0,
    0,
    449,
    450,
    1,
    0,
    0,
    0,
    450,
    451,
    1,
    0,
    0,
    0,
    451,
    453,
    5,
    24,
    0,
    0,
    452,
    447,
    1,
    0,
    0,
    0,
    452,
    453,
    1,
    0,
    0,
    0,
    453,
    47,
    1,
    0,
    0,
    0,
    454,
    455,
    3,
    50,
    25,
    0,
    455,
    456,
    3,
    62,
    31,
    0,
    456,
    458,
    3,
    54,
    27,
    0,
    457,
    459,
    3,
    52,
    26,
    0,
    458,
    457,
    1,
    0,
    0,
    0,
    458,
    459,
    1,
    0,
    0,
    0,
    459,
    462,
    1,
    0,
    0,
    0,
    460,
    463,
    5,
    2,
    0,
    0,
    461,
    463,
    3,
    94,
    47,
    0,
    462,
    460,
    1,
    0,
    0,
    0,
    462,
    461,
    1,
    0,
    0,
    0,
    463,
    49,
    1,
    0,
    0,
    0,
    464,
    466,
    5,
    38,
    0,
    0,
    465,
    467,
    3,
    194,
    97,
    0,
    466,
    465,
    1,
    0,
    0,
    0,
    466,
    467,
    1,
    0,
    0,
    0,
    467,
    472,
    1,
    0,
    0,
    0,
    468,
    472,
    5,
    125,
    0,
    0,
    469,
    472,
    5,
    126,
    0,
    0,
    470,
    472,
    5,
    127,
    0,
    0,
    471,
    464,
    1,
    0,
    0,
    0,
    471,
    468,
    1,
    0,
    0,
    0,
    471,
    469,
    1,
    0,
    0,
    0,
    471,
    470,
    1,
    0,
    0,
    0,
    472,
    51,
    1,
    0,
    0,
    0,
    473,
    474,
    5,
    39,
    0,
    0,
    474,
    475,
    3,
    62,
    31,
    0,
    475,
    53,
    1,
    0,
    0,
    0,
    476,
    485,
    5,
    114,
    0,
    0,
    477,
    485,
    5,
    119,
    0,
    0,
    478,
    485,
    5,
    116,
    0,
    0,
    479,
    485,
    5,
    118,
    0,
    0,
    480,
    485,
    5,
    120,
    0,
    0,
    481,
    485,
    3,
    92,
    46,
    0,
    482,
    485,
    3,
    46,
    23,
    0,
    483,
    485,
    3,
    198,
    99,
    0,
    484,
    476,
    1,
    0,
    0,
    0,
    484,
    477,
    1,
    0,
    0,
    0,
    484,
    478,
    1,
    0,
    0,
    0,
    484,
    479,
    1,
    0,
    0,
    0,
    484,
    480,
    1,
    0,
    0,
    0,
    484,
    481,
    1,
    0,
    0,
    0,
    484,
    482,
    1,
    0,
    0,
    0,
    484,
    483,
    1,
    0,
    0,
    0,
    485,
    488,
    1,
    0,
    0,
    0,
    486,
    484,
    1,
    0,
    0,
    0,
    486,
    487,
    1,
    0,
    0,
    0,
    487,
    55,
    1,
    0,
    0,
    0,
    488,
    486,
    1,
    0,
    0,
    0,
    489,
    490,
    5,
    40,
    0,
    0,
    490,
    491,
    3,
    194,
    97,
    0,
    491,
    493,
    3,
    66,
    33,
    0,
    492,
    494,
    5,
    108,
    0,
    0,
    493,
    492,
    1,
    0,
    0,
    0,
    493,
    494,
    1,
    0,
    0,
    0,
    494,
    495,
    1,
    0,
    0,
    0,
    495,
    496,
    5,
    2,
    0,
    0,
    496,
    57,
    1,
    0,
    0,
    0,
    497,
    498,
    3,
    194,
    97,
    0,
    498,
    59,
    1,
    0,
    0,
    0,
    499,
    500,
    5,
    41,
    0,
    0,
    500,
    501,
    3,
    194,
    97,
    0,
    501,
    503,
    5,
    15,
    0,
    0,
    502,
    504,
    3,
    58,
    29,
    0,
    503,
    502,
    1,
    0,
    0,
    0,
    503,
    504,
    1,
    0,
    0,
    0,
    504,
    509,
    1,
    0,
    0,
    0,
    505,
    506,
    5,
    16,
    0,
    0,
    506,
    508,
    3,
    58,
    29,
    0,
    507,
    505,
    1,
    0,
    0,
    0,
    508,
    511,
    1,
    0,
    0,
    0,
    509,
    507,
    1,
    0,
    0,
    0,
    509,
    510,
    1,
    0,
    0,
    0,
    510,
    512,
    1,
    0,
    0,
    0,
    511,
    509,
    1,
    0,
    0,
    0,
    512,
    513,
    5,
    17,
    0,
    0,
    513,
    61,
    1,
    0,
    0,
    0,
    514,
    523,
    5,
    23,
    0,
    0,
    515,
    520,
    3,
    64,
    32,
    0,
    516,
    517,
    5,
    16,
    0,
    0,
    517,
    519,
    3,
    64,
    32,
    0,
    518,
    516,
    1,
    0,
    0,
    0,
    519,
    522,
    1,
    0,
    0,
    0,
    520,
    518,
    1,
    0,
    0,
    0,
    520,
    521,
    1,
    0,
    0,
    0,
    521,
    524,
    1,
    0,
    0,
    0,
    522,
    520,
    1,
    0,
    0,
    0,
    523,
    515,
    1,
    0,
    0,
    0,
    523,
    524,
    1,
    0,
    0,
    0,
    524,
    525,
    1,
    0,
    0,
    0,
    525,
    526,
    5,
    24,
    0,
    0,
    526,
    63,
    1,
    0,
    0,
    0,
    527,
    529,
    3,
    76,
    38,
    0,
    528,
    530,
    3,
    90,
    45,
    0,
    529,
    528,
    1,
    0,
    0,
    0,
    529,
    530,
    1,
    0,
    0,
    0,
    530,
    532,
    1,
    0,
    0,
    0,
    531,
    533,
    3,
    194,
    97,
    0,
    532,
    531,
    1,
    0,
    0,
    0,
    532,
    533,
    1,
    0,
    0,
    0,
    533,
    65,
    1,
    0,
    0,
    0,
    534,
    543,
    5,
    23,
    0,
    0,
    535,
    540,
    3,
    68,
    34,
    0,
    536,
    537,
    5,
    16,
    0,
    0,
    537,
    539,
    3,
    68,
    34,
    0,
    538,
    536,
    1,
    0,
    0,
    0,
    539,
    542,
    1,
    0,
    0,
    0,
    540,
    538,
    1,
    0,
    0,
    0,
    540,
    541,
    1,
    0,
    0,
    0,
    541,
    544,
    1,
    0,
    0,
    0,
    542,
    540,
    1,
    0,
    0,
    0,
    543,
    535,
    1,
    0,
    0,
    0,
    543,
    544,
    1,
    0,
    0,
    0,
    544,
    545,
    1,
    0,
    0,
    0,
    545,
    546,
    5,
    24,
    0,
    0,
    546,
    67,
    1,
    0,
    0,
    0,
    547,
    549,
    3,
    76,
    38,
    0,
    548,
    550,
    5,
    115,
    0,
    0,
    549,
    548,
    1,
    0,
    0,
    0,
    549,
    550,
    1,
    0,
    0,
    0,
    550,
    552,
    1,
    0,
    0,
    0,
    551,
    553,
    3,
    194,
    97,
    0,
    552,
    551,
    1,
    0,
    0,
    0,
    552,
    553,
    1,
    0,
    0,
    0,
    553,
    69,
    1,
    0,
    0,
    0,
    554,
    563,
    5,
    23,
    0,
    0,
    555,
    560,
    3,
    72,
    36,
    0,
    556,
    557,
    5,
    16,
    0,
    0,
    557,
    559,
    3,
    72,
    36,
    0,
    558,
    556,
    1,
    0,
    0,
    0,
    559,
    562,
    1,
    0,
    0,
    0,
    560,
    558,
    1,
    0,
    0,
    0,
    560,
    561,
    1,
    0,
    0,
    0,
    561,
    564,
    1,
    0,
    0,
    0,
    562,
    560,
    1,
    0,
    0,
    0,
    563,
    555,
    1,
    0,
    0,
    0,
    563,
    564,
    1,
    0,
    0,
    0,
    564,
    565,
    1,
    0,
    0,
    0,
    565,
    566,
    5,
    24,
    0,
    0,
    566,
    71,
    1,
    0,
    0,
    0,
    567,
    569,
    3,
    76,
    38,
    0,
    568,
    570,
    3,
    90,
    45,
    0,
    569,
    568,
    1,
    0,
    0,
    0,
    569,
    570,
    1,
    0,
    0,
    0,
    570,
    73,
    1,
    0,
    0,
    0,
    571,
    573,
    3,
    76,
    38,
    0,
    572,
    574,
    3,
    90,
    45,
    0,
    573,
    572,
    1,
    0,
    0,
    0,
    573,
    574,
    1,
    0,
    0,
    0,
    574,
    575,
    1,
    0,
    0,
    0,
    575,
    576,
    3,
    194,
    97,
    0,
    576,
    75,
    1,
    0,
    0,
    0,
    577,
    578,
    6,
    38,
    -1,
    0,
    578,
    585,
    3,
    138,
    69,
    0,
    579,
    585,
    3,
    78,
    39,
    0,
    580,
    585,
    3,
    82,
    41,
    0,
    581,
    585,
    3,
    88,
    44,
    0,
    582,
    583,
    5,
    44,
    0,
    0,
    583,
    585,
    5,
    117,
    0,
    0,
    584,
    577,
    1,
    0,
    0,
    0,
    584,
    579,
    1,
    0,
    0,
    0,
    584,
    580,
    1,
    0,
    0,
    0,
    584,
    581,
    1,
    0,
    0,
    0,
    584,
    582,
    1,
    0,
    0,
    0,
    585,
    594,
    1,
    0,
    0,
    0,
    586,
    587,
    10,
    3,
    0,
    0,
    587,
    589,
    5,
    42,
    0,
    0,
    588,
    590,
    3,
    140,
    70,
    0,
    589,
    588,
    1,
    0,
    0,
    0,
    589,
    590,
    1,
    0,
    0,
    0,
    590,
    591,
    1,
    0,
    0,
    0,
    591,
    593,
    5,
    43,
    0,
    0,
    592,
    586,
    1,
    0,
    0,
    0,
    593,
    596,
    1,
    0,
    0,
    0,
    594,
    592,
    1,
    0,
    0,
    0,
    594,
    595,
    1,
    0,
    0,
    0,
    595,
    77,
    1,
    0,
    0,
    0,
    596,
    594,
    1,
    0,
    0,
    0,
    597,
    602,
    3,
    194,
    97,
    0,
    598,
    599,
    5,
    45,
    0,
    0,
    599,
    601,
    3,
    194,
    97,
    0,
    600,
    598,
    1,
    0,
    0,
    0,
    601,
    604,
    1,
    0,
    0,
    0,
    602,
    600,
    1,
    0,
    0,
    0,
    602,
    603,
    1,
    0,
    0,
    0,
    603,
    79,
    1,
    0,
    0,
    0,
    604,
    602,
    1,
    0,
    0,
    0,
    605,
    608,
    3,
    138,
    69,
    0,
    606,
    608,
    3,
    78,
    39,
    0,
    607,
    605,
    1,
    0,
    0,
    0,
    607,
    606,
    1,
    0,
    0,
    0,
    608,
    81,
    1,
    0,
    0,
    0,
    609,
    610,
    5,
    46,
    0,
    0,
    610,
    611,
    5,
    23,
    0,
    0,
    611,
    613,
    3,
    80,
    40,
    0,
    612,
    614,
    3,
    84,
    42,
    0,
    613,
    612,
    1,
    0,
    0,
    0,
    613,
    614,
    1,
    0,
    0,
    0,
    614,
    615,
    1,
    0,
    0,
    0,
    615,
    616,
    5,
    47,
    0,
    0,
    616,
    618,
    3,
    76,
    38,
    0,
    617,
    619,
    3,
    86,
    43,
    0,
    618,
    617,
    1,
    0,
    0,
    0,
    618,
    619,
    1,
    0,
    0,
    0,
    619,
    620,
    1,
    0,
    0,
    0,
    620,
    621,
    5,
    24,
    0,
    0,
    621,
    83,
    1,
    0,
    0,
    0,
    622,
    623,
    3,
    194,
    97,
    0,
    623,
    85,
    1,
    0,
    0,
    0,
    624,
    625,
    3,
    194,
    97,
    0,
    625,
    87,
    1,
    0,
    0,
    0,
    626,
    627,
    5,
    38,
    0,
    0,
    627,
    633,
    3,
    70,
    35,
    0,
    628,
    632,
    5,
    116,
    0,
    0,
    629,
    632,
    5,
    114,
    0,
    0,
    630,
    632,
    3,
    92,
    46,
    0,
    631,
    628,
    1,
    0,
    0,
    0,
    631,
    629,
    1,
    0,
    0,
    0,
    631,
    630,
    1,
    0,
    0,
    0,
    632,
    635,
    1,
    0,
    0,
    0,
    633,
    631,
    1,
    0,
    0,
    0,
    633,
    634,
    1,
    0,
    0,
    0,
    634,
    638,
    1,
    0,
    0,
    0,
    635,
    633,
    1,
    0,
    0,
    0,
    636,
    637,
    5,
    39,
    0,
    0,
    637,
    639,
    3,
    70,
    35,
    0,
    638,
    636,
    1,
    0,
    0,
    0,
    638,
    639,
    1,
    0,
    0,
    0,
    639,
    89,
    1,
    0,
    0,
    0,
    640,
    641,
    7,
    3,
    0,
    0,
    641,
    91,
    1,
    0,
    0,
    0,
    642,
    643,
    7,
    4,
    0,
    0,
    643,
    93,
    1,
    0,
    0,
    0,
    644,
    648,
    5,
    15,
    0,
    0,
    645,
    647,
    3,
    96,
    48,
    0,
    646,
    645,
    1,
    0,
    0,
    0,
    647,
    650,
    1,
    0,
    0,
    0,
    648,
    646,
    1,
    0,
    0,
    0,
    648,
    649,
    1,
    0,
    0,
    0,
    649,
    651,
    1,
    0,
    0,
    0,
    650,
    648,
    1,
    0,
    0,
    0,
    651,
    652,
    5,
    17,
    0,
    0,
    652,
    95,
    1,
    0,
    0,
    0,
    653,
    669,
    3,
    100,
    50,
    0,
    654,
    669,
    3,
    102,
    51,
    0,
    655,
    669,
    3,
    106,
    53,
    0,
    656,
    669,
    3,
    112,
    56,
    0,
    657,
    669,
    3,
    94,
    47,
    0,
    658,
    669,
    3,
    114,
    57,
    0,
    659,
    669,
    3,
    118,
    59,
    0,
    660,
    669,
    3,
    120,
    60,
    0,
    661,
    669,
    3,
    122,
    61,
    0,
    662,
    669,
    3,
    124,
    62,
    0,
    663,
    669,
    3,
    126,
    63,
    0,
    664,
    669,
    3,
    128,
    64,
    0,
    665,
    669,
    3,
    108,
    54,
    0,
    666,
    669,
    3,
    110,
    55,
    0,
    667,
    669,
    3,
    130,
    65,
    0,
    668,
    653,
    1,
    0,
    0,
    0,
    668,
    654,
    1,
    0,
    0,
    0,
    668,
    655,
    1,
    0,
    0,
    0,
    668,
    656,
    1,
    0,
    0,
    0,
    668,
    657,
    1,
    0,
    0,
    0,
    668,
    658,
    1,
    0,
    0,
    0,
    668,
    659,
    1,
    0,
    0,
    0,
    668,
    660,
    1,
    0,
    0,
    0,
    668,
    661,
    1,
    0,
    0,
    0,
    668,
    662,
    1,
    0,
    0,
    0,
    668,
    663,
    1,
    0,
    0,
    0,
    668,
    664,
    1,
    0,
    0,
    0,
    668,
    665,
    1,
    0,
    0,
    0,
    668,
    666,
    1,
    0,
    0,
    0,
    668,
    667,
    1,
    0,
    0,
    0,
    669,
    97,
    1,
    0,
    0,
    0,
    670,
    671,
    3,
    140,
    70,
    0,
    671,
    672,
    5,
    2,
    0,
    0,
    672,
    99,
    1,
    0,
    0,
    0,
    673,
    674,
    5,
    51,
    0,
    0,
    674,
    675,
    5,
    23,
    0,
    0,
    675,
    676,
    3,
    140,
    70,
    0,
    676,
    677,
    5,
    24,
    0,
    0,
    677,
    680,
    3,
    96,
    48,
    0,
    678,
    679,
    5,
    52,
    0,
    0,
    679,
    681,
    3,
    96,
    48,
    0,
    680,
    678,
    1,
    0,
    0,
    0,
    680,
    681,
    1,
    0,
    0,
    0,
    681,
    101,
    1,
    0,
    0,
    0,
    682,
    683,
    5,
    53,
    0,
    0,
    683,
    685,
    3,
    140,
    70,
    0,
    684,
    686,
    3,
    52,
    26,
    0,
    685,
    684,
    1,
    0,
    0,
    0,
    685,
    686,
    1,
    0,
    0,
    0,
    686,
    687,
    1,
    0,
    0,
    0,
    687,
    689,
    3,
    94,
    47,
    0,
    688,
    690,
    3,
    104,
    52,
    0,
    689,
    688,
    1,
    0,
    0,
    0,
    690,
    691,
    1,
    0,
    0,
    0,
    691,
    689,
    1,
    0,
    0,
    0,
    691,
    692,
    1,
    0,
    0,
    0,
    692,
    103,
    1,
    0,
    0,
    0,
    693,
    698,
    5,
    54,
    0,
    0,
    694,
    696,
    3,
    194,
    97,
    0,
    695,
    694,
    1,
    0,
    0,
    0,
    695,
    696,
    1,
    0,
    0,
    0,
    696,
    697,
    1,
    0,
    0,
    0,
    697,
    699,
    3,
    62,
    31,
    0,
    698,
    695,
    1,
    0,
    0,
    0,
    698,
    699,
    1,
    0,
    0,
    0,
    699,
    700,
    1,
    0,
    0,
    0,
    700,
    701,
    3,
    94,
    47,
    0,
    701,
    105,
    1,
    0,
    0,
    0,
    702,
    703,
    5,
    55,
    0,
    0,
    703,
    704,
    5,
    23,
    0,
    0,
    704,
    705,
    3,
    140,
    70,
    0,
    705,
    706,
    5,
    24,
    0,
    0,
    706,
    707,
    3,
    96,
    48,
    0,
    707,
    107,
    1,
    0,
    0,
    0,
    708,
    711,
    3,
    132,
    66,
    0,
    709,
    711,
    3,
    98,
    49,
    0,
    710,
    708,
    1,
    0,
    0,
    0,
    710,
    709,
    1,
    0,
    0,
    0,
    711,
    109,
    1,
    0,
    0,
    0,
    712,
    713,
    5,
    56,
    0,
    0,
    713,
    714,
    3,
    94,
    47,
    0,
    714,
    111,
    1,
    0,
    0,
    0,
    715,
    716,
    5,
    27,
    0,
    0,
    716,
    719,
    5,
    23,
    0,
    0,
    717,
    720,
    3,
    108,
    54,
    0,
    718,
    720,
    5,
    2,
    0,
    0,
    719,
    717,
    1,
    0,
    0,
    0,
    719,
    718,
    1,
    0,
    0,
    0,
    720,
    723,
    1,
    0,
    0,
    0,
    721,
    724,
    3,
    98,
    49,
    0,
    722,
    724,
    5,
    2,
    0,
    0,
    723,
    721,
    1,
    0,
    0,
    0,
    723,
    722,
    1,
    0,
    0,
    0,
    724,
    726,
    1,
    0,
    0,
    0,
    725,
    727,
    3,
    140,
    70,
    0,
    726,
    725,
    1,
    0,
    0,
    0,
    726,
    727,
    1,
    0,
    0,
    0,
    727,
    728,
    1,
    0,
    0,
    0,
    728,
    729,
    5,
    24,
    0,
    0,
    729,
    730,
    3,
    96,
    48,
    0,
    730,
    113,
    1,
    0,
    0,
    0,
    731,
    733,
    5,
    57,
    0,
    0,
    732,
    734,
    5,
    129,
    0,
    0,
    733,
    732,
    1,
    0,
    0,
    0,
    733,
    734,
    1,
    0,
    0,
    0,
    734,
    739,
    1,
    0,
    0,
    0,
    735,
    736,
    5,
    23,
    0,
    0,
    736,
    737,
    3,
    116,
    58,
    0,
    737,
    738,
    5,
    24,
    0,
    0,
    738,
    740,
    1,
    0,
    0,
    0,
    739,
    735,
    1,
    0,
    0,
    0,
    739,
    740,
    1,
    0,
    0,
    0,
    740,
    741,
    1,
    0,
    0,
    0,
    741,
    742,
    3,
    154,
    77,
    0,
    742,
    115,
    1,
    0,
    0,
    0,
    743,
    744,
    3,
    200,
    100,
    0,
    744,
    117,
    1,
    0,
    0,
    0,
    745,
    746,
    5,
    58,
    0,
    0,
    746,
    747,
    3,
    96,
    48,
    0,
    747,
    748,
    5,
    55,
    0,
    0,
    748,
    749,
    5,
    23,
    0,
    0,
    749,
    750,
    3,
    140,
    70,
    0,
    750,
    751,
    5,
    24,
    0,
    0,
    751,
    752,
    5,
    2,
    0,
    0,
    752,
    119,
    1,
    0,
    0,
    0,
    753,
    754,
    5,
    112,
    0,
    0,
    754,
    755,
    5,
    2,
    0,
    0,
    755,
    121,
    1,
    0,
    0,
    0,
    756,
    757,
    5,
    109,
    0,
    0,
    757,
    758,
    5,
    2,
    0,
    0,
    758,
    123,
    1,
    0,
    0,
    0,
    759,
    761,
    5,
    59,
    0,
    0,
    760,
    762,
    3,
    140,
    70,
    0,
    761,
    760,
    1,
    0,
    0,
    0,
    761,
    762,
    1,
    0,
    0,
    0,
    762,
    763,
    1,
    0,
    0,
    0,
    763,
    764,
    5,
    2,
    0,
    0,
    764,
    125,
    1,
    0,
    0,
    0,
    765,
    766,
    5,
    60,
    0,
    0,
    766,
    767,
    5,
    2,
    0,
    0,
    767,
    127,
    1,
    0,
    0,
    0,
    768,
    769,
    5,
    61,
    0,
    0,
    769,
    770,
    3,
    152,
    76,
    0,
    770,
    771,
    5,
    2,
    0,
    0,
    771,
    129,
    1,
    0,
    0,
    0,
    772,
    773,
    5,
    62,
    0,
    0,
    773,
    774,
    3,
    152,
    76,
    0,
    774,
    775,
    5,
    2,
    0,
    0,
    775,
    131,
    1,
    0,
    0,
    0,
    776,
    777,
    5,
    63,
    0,
    0,
    777,
    784,
    3,
    136,
    68,
    0,
    778,
    784,
    3,
    74,
    37,
    0,
    779,
    780,
    5,
    23,
    0,
    0,
    780,
    781,
    3,
    134,
    67,
    0,
    781,
    782,
    5,
    24,
    0,
    0,
    782,
    784,
    1,
    0,
    0,
    0,
    783,
    776,
    1,
    0,
    0,
    0,
    783,
    778,
    1,
    0,
    0,
    0,
    783,
    779,
    1,
    0,
    0,
    0,
    784,
    787,
    1,
    0,
    0,
    0,
    785,
    786,
    5,
    11,
    0,
    0,
    786,
    788,
    3,
    140,
    70,
    0,
    787,
    785,
    1,
    0,
    0,
    0,
    787,
    788,
    1,
    0,
    0,
    0,
    788,
    789,
    1,
    0,
    0,
    0,
    789,
    790,
    5,
    2,
    0,
    0,
    790,
    133,
    1,
    0,
    0,
    0,
    791,
    793,
    3,
    74,
    37,
    0,
    792,
    791,
    1,
    0,
    0,
    0,
    792,
    793,
    1,
    0,
    0,
    0,
    793,
    800,
    1,
    0,
    0,
    0,
    794,
    796,
    5,
    16,
    0,
    0,
    795,
    797,
    3,
    74,
    37,
    0,
    796,
    795,
    1,
    0,
    0,
    0,
    796,
    797,
    1,
    0,
    0,
    0,
    797,
    799,
    1,
    0,
    0,
    0,
    798,
    794,
    1,
    0,
    0,
    0,
    799,
    802,
    1,
    0,
    0,
    0,
    800,
    798,
    1,
    0,
    0,
    0,
    800,
    801,
    1,
    0,
    0,
    0,
    801,
    135,
    1,
    0,
    0,
    0,
    802,
    800,
    1,
    0,
    0,
    0,
    803,
    810,
    5,
    23,
    0,
    0,
    804,
    806,
    3,
    194,
    97,
    0,
    805,
    804,
    1,
    0,
    0,
    0,
    805,
    806,
    1,
    0,
    0,
    0,
    806,
    807,
    1,
    0,
    0,
    0,
    807,
    809,
    5,
    16,
    0,
    0,
    808,
    805,
    1,
    0,
    0,
    0,
    809,
    812,
    1,
    0,
    0,
    0,
    810,
    808,
    1,
    0,
    0,
    0,
    810,
    811,
    1,
    0,
    0,
    0,
    811,
    814,
    1,
    0,
    0,
    0,
    812,
    810,
    1,
    0,
    0,
    0,
    813,
    815,
    3,
    194,
    97,
    0,
    814,
    813,
    1,
    0,
    0,
    0,
    814,
    815,
    1,
    0,
    0,
    0,
    815,
    816,
    1,
    0,
    0,
    0,
    816,
    817,
    5,
    24,
    0,
    0,
    817,
    137,
    1,
    0,
    0,
    0,
    818,
    819,
    7,
    5,
    0,
    0,
    819,
    139,
    1,
    0,
    0,
    0,
    820,
    821,
    6,
    70,
    -1,
    0,
    821,
    822,
    5,
    69,
    0,
    0,
    822,
    839,
    3,
    76,
    38,
    0,
    823,
    824,
    5,
    23,
    0,
    0,
    824,
    825,
    3,
    140,
    70,
    0,
    825,
    826,
    5,
    24,
    0,
    0,
    826,
    839,
    1,
    0,
    0,
    0,
    827,
    828,
    7,
    6,
    0,
    0,
    828,
    839,
    3,
    140,
    70,
    19,
    829,
    830,
    7,
    7,
    0,
    0,
    830,
    839,
    3,
    140,
    70,
    18,
    831,
    832,
    5,
    71,
    0,
    0,
    832,
    839,
    3,
    140,
    70,
    17,
    833,
    834,
    5,
    72,
    0,
    0,
    834,
    839,
    3,
    140,
    70,
    16,
    835,
    836,
    5,
    6,
    0,
    0,
    836,
    839,
    3,
    140,
    70,
    15,
    837,
    839,
    3,
    142,
    71,
    0,
    838,
    820,
    1,
    0,
    0,
    0,
    838,
    823,
    1,
    0,
    0,
    0,
    838,
    827,
    1,
    0,
    0,
    0,
    838,
    829,
    1,
    0,
    0,
    0,
    838,
    831,
    1,
    0,
    0,
    0,
    838,
    833,
    1,
    0,
    0,
    0,
    838,
    835,
    1,
    0,
    0,
    0,
    838,
    837,
    1,
    0,
    0,
    0,
    839,
    914,
    1,
    0,
    0,
    0,
    840,
    841,
    10,
    14,
    0,
    0,
    841,
    842,
    5,
    73,
    0,
    0,
    842,
    913,
    3,
    140,
    70,
    14,
    843,
    844,
    10,
    13,
    0,
    0,
    844,
    845,
    7,
    8,
    0,
    0,
    845,
    913,
    3,
    140,
    70,
    14,
    846,
    847,
    10,
    12,
    0,
    0,
    847,
    848,
    7,
    7,
    0,
    0,
    848,
    913,
    3,
    140,
    70,
    13,
    849,
    850,
    10,
    11,
    0,
    0,
    850,
    851,
    7,
    9,
    0,
    0,
    851,
    913,
    3,
    140,
    70,
    12,
    852,
    853,
    10,
    10,
    0,
    0,
    853,
    854,
    5,
    29,
    0,
    0,
    854,
    913,
    3,
    140,
    70,
    11,
    855,
    856,
    10,
    9,
    0,
    0,
    856,
    857,
    5,
    5,
    0,
    0,
    857,
    913,
    3,
    140,
    70,
    10,
    858,
    859,
    10,
    8,
    0,
    0,
    859,
    860,
    5,
    28,
    0,
    0,
    860,
    913,
    3,
    140,
    70,
    9,
    861,
    862,
    10,
    7,
    0,
    0,
    862,
    863,
    7,
    10,
    0,
    0,
    863,
    913,
    3,
    140,
    70,
    8,
    864,
    865,
    10,
    6,
    0,
    0,
    865,
    866,
    7,
    11,
    0,
    0,
    866,
    913,
    3,
    140,
    70,
    7,
    867,
    868,
    10,
    5,
    0,
    0,
    868,
    869,
    5,
    76,
    0,
    0,
    869,
    913,
    3,
    140,
    70,
    6,
    870,
    871,
    10,
    4,
    0,
    0,
    871,
    872,
    5,
    4,
    0,
    0,
    872,
    913,
    3,
    140,
    70,
    5,
    873,
    874,
    10,
    3,
    0,
    0,
    874,
    875,
    5,
    77,
    0,
    0,
    875,
    876,
    3,
    140,
    70,
    0,
    876,
    877,
    5,
    70,
    0,
    0,
    877,
    878,
    3,
    140,
    70,
    3,
    878,
    913,
    1,
    0,
    0,
    0,
    879,
    880,
    10,
    2,
    0,
    0,
    880,
    881,
    7,
    12,
    0,
    0,
    881,
    913,
    3,
    140,
    70,
    3,
    882,
    883,
    10,
    27,
    0,
    0,
    883,
    913,
    7,
    6,
    0,
    0,
    884,
    885,
    10,
    25,
    0,
    0,
    885,
    886,
    5,
    42,
    0,
    0,
    886,
    887,
    3,
    140,
    70,
    0,
    887,
    888,
    5,
    43,
    0,
    0,
    888,
    913,
    1,
    0,
    0,
    0,
    889,
    890,
    10,
    24,
    0,
    0,
    890,
    892,
    5,
    42,
    0,
    0,
    891,
    893,
    3,
    140,
    70,
    0,
    892,
    891,
    1,
    0,
    0,
    0,
    892,
    893,
    1,
    0,
    0,
    0,
    893,
    894,
    1,
    0,
    0,
    0,
    894,
    896,
    5,
    70,
    0,
    0,
    895,
    897,
    3,
    140,
    70,
    0,
    896,
    895,
    1,
    0,
    0,
    0,
    896,
    897,
    1,
    0,
    0,
    0,
    897,
    898,
    1,
    0,
    0,
    0,
    898,
    913,
    5,
    43,
    0,
    0,
    899,
    900,
    10,
    23,
    0,
    0,
    900,
    901,
    5,
    45,
    0,
    0,
    901,
    913,
    3,
    194,
    97,
    0,
    902,
    903,
    10,
    22,
    0,
    0,
    903,
    904,
    5,
    15,
    0,
    0,
    904,
    905,
    3,
    146,
    73,
    0,
    905,
    906,
    5,
    17,
    0,
    0,
    906,
    913,
    1,
    0,
    0,
    0,
    907,
    908,
    10,
    21,
    0,
    0,
    908,
    909,
    5,
    23,
    0,
    0,
    909,
    910,
    3,
    150,
    75,
    0,
    910,
    911,
    5,
    24,
    0,
    0,
    911,
    913,
    1,
    0,
    0,
    0,
    912,
    840,
    1,
    0,
    0,
    0,
    912,
    843,
    1,
    0,
    0,
    0,
    912,
    846,
    1,
    0,
    0,
    0,
    912,
    849,
    1,
    0,
    0,
    0,
    912,
    852,
    1,
    0,
    0,
    0,
    912,
    855,
    1,
    0,
    0,
    0,
    912,
    858,
    1,
    0,
    0,
    0,
    912,
    861,
    1,
    0,
    0,
    0,
    912,
    864,
    1,
    0,
    0,
    0,
    912,
    867,
    1,
    0,
    0,
    0,
    912,
    870,
    1,
    0,
    0,
    0,
    912,
    873,
    1,
    0,
    0,
    0,
    912,
    879,
    1,
    0,
    0,
    0,
    912,
    882,
    1,
    0,
    0,
    0,
    912,
    884,
    1,
    0,
    0,
    0,
    912,
    889,
    1,
    0,
    0,
    0,
    912,
    899,
    1,
    0,
    0,
    0,
    912,
    902,
    1,
    0,
    0,
    0,
    912,
    907,
    1,
    0,
    0,
    0,
    913,
    916,
    1,
    0,
    0,
    0,
    914,
    912,
    1,
    0,
    0,
    0,
    914,
    915,
    1,
    0,
    0,
    0,
    915,
    141,
    1,
    0,
    0,
    0,
    916,
    914,
    1,
    0,
    0,
    0,
    917,
    927,
    5,
    102,
    0,
    0,
    918,
    927,
    3,
    192,
    96,
    0,
    919,
    927,
    3,
    196,
    98,
    0,
    920,
    927,
    3,
    200,
    100,
    0,
    921,
    927,
    3,
    194,
    97,
    0,
    922,
    927,
    5,
    122,
    0,
    0,
    923,
    927,
    5,
    117,
    0,
    0,
    924,
    927,
    3,
    190,
    95,
    0,
    925,
    927,
    3,
    76,
    38,
    0,
    926,
    917,
    1,
    0,
    0,
    0,
    926,
    918,
    1,
    0,
    0,
    0,
    926,
    919,
    1,
    0,
    0,
    0,
    926,
    920,
    1,
    0,
    0,
    0,
    926,
    921,
    1,
    0,
    0,
    0,
    926,
    922,
    1,
    0,
    0,
    0,
    926,
    923,
    1,
    0,
    0,
    0,
    926,
    924,
    1,
    0,
    0,
    0,
    926,
    925,
    1,
    0,
    0,
    0,
    927,
    143,
    1,
    0,
    0,
    0,
    928,
    933,
    3,
    140,
    70,
    0,
    929,
    930,
    5,
    16,
    0,
    0,
    930,
    932,
    3,
    140,
    70,
    0,
    931,
    929,
    1,
    0,
    0,
    0,
    932,
    935,
    1,
    0,
    0,
    0,
    933,
    931,
    1,
    0,
    0,
    0,
    933,
    934,
    1,
    0,
    0,
    0,
    934,
    145,
    1,
    0,
    0,
    0,
    935,
    933,
    1,
    0,
    0,
    0,
    936,
    941,
    3,
    148,
    74,
    0,
    937,
    938,
    5,
    16,
    0,
    0,
    938,
    940,
    3,
    148,
    74,
    0,
    939,
    937,
    1,
    0,
    0,
    0,
    940,
    943,
    1,
    0,
    0,
    0,
    941,
    939,
    1,
    0,
    0,
    0,
    941,
    942,
    1,
    0,
    0,
    0,
    942,
    945,
    1,
    0,
    0,
    0,
    943,
    941,
    1,
    0,
    0,
    0,
    944,
    946,
    5,
    16,
    0,
    0,
    945,
    944,
    1,
    0,
    0,
    0,
    945,
    946,
    1,
    0,
    0,
    0,
    946,
    147,
    1,
    0,
    0,
    0,
    947,
    948,
    3,
    194,
    97,
    0,
    948,
    949,
    5,
    70,
    0,
    0,
    949,
    950,
    3,
    140,
    70,
    0,
    950,
    149,
    1,
    0,
    0,
    0,
    951,
    953,
    5,
    15,
    0,
    0,
    952,
    954,
    3,
    146,
    73,
    0,
    953,
    952,
    1,
    0,
    0,
    0,
    953,
    954,
    1,
    0,
    0,
    0,
    954,
    955,
    1,
    0,
    0,
    0,
    955,
    960,
    5,
    17,
    0,
    0,
    956,
    958,
    3,
    144,
    72,
    0,
    957,
    956,
    1,
    0,
    0,
    0,
    957,
    958,
    1,
    0,
    0,
    0,
    958,
    960,
    1,
    0,
    0,
    0,
    959,
    951,
    1,
    0,
    0,
    0,
    959,
    957,
    1,
    0,
    0,
    0,
    960,
    151,
    1,
    0,
    0,
    0,
    961,
    962,
    3,
    140,
    70,
    0,
    962,
    963,
    5,
    23,
    0,
    0,
    963,
    964,
    3,
    150,
    75,
    0,
    964,
    965,
    5,
    24,
    0,
    0,
    965,
    153,
    1,
    0,
    0,
    0,
    966,
    970,
    5,
    15,
    0,
    0,
    967,
    969,
    3,
    156,
    78,
    0,
    968,
    967,
    1,
    0,
    0,
    0,
    969,
    972,
    1,
    0,
    0,
    0,
    970,
    968,
    1,
    0,
    0,
    0,
    970,
    971,
    1,
    0,
    0,
    0,
    971,
    973,
    1,
    0,
    0,
    0,
    972,
    970,
    1,
    0,
    0,
    0,
    973,
    974,
    5,
    17,
    0,
    0,
    974,
    155,
    1,
    0,
    0,
    0,
    975,
    993,
    3,
    194,
    97,
    0,
    976,
    993,
    3,
    154,
    77,
    0,
    977,
    993,
    3,
    158,
    79,
    0,
    978,
    993,
    3,
    164,
    82,
    0,
    979,
    993,
    3,
    166,
    83,
    0,
    980,
    993,
    3,
    172,
    86,
    0,
    981,
    993,
    3,
    174,
    87,
    0,
    982,
    993,
    3,
    176,
    88,
    0,
    983,
    993,
    3,
    180,
    90,
    0,
    984,
    993,
    3,
    184,
    92,
    0,
    985,
    993,
    3,
    186,
    93,
    0,
    986,
    993,
    5,
    109,
    0,
    0,
    987,
    993,
    5,
    112,
    0,
    0,
    988,
    993,
    5,
    113,
    0,
    0,
    989,
    993,
    3,
    192,
    96,
    0,
    990,
    993,
    3,
    200,
    100,
    0,
    991,
    993,
    3,
    196,
    98,
    0,
    992,
    975,
    1,
    0,
    0,
    0,
    992,
    976,
    1,
    0,
    0,
    0,
    992,
    977,
    1,
    0,
    0,
    0,
    992,
    978,
    1,
    0,
    0,
    0,
    992,
    979,
    1,
    0,
    0,
    0,
    992,
    980,
    1,
    0,
    0,
    0,
    992,
    981,
    1,
    0,
    0,
    0,
    992,
    982,
    1,
    0,
    0,
    0,
    992,
    983,
    1,
    0,
    0,
    0,
    992,
    984,
    1,
    0,
    0,
    0,
    992,
    985,
    1,
    0,
    0,
    0,
    992,
    986,
    1,
    0,
    0,
    0,
    992,
    987,
    1,
    0,
    0,
    0,
    992,
    988,
    1,
    0,
    0,
    0,
    992,
    989,
    1,
    0,
    0,
    0,
    992,
    990,
    1,
    0,
    0,
    0,
    992,
    991,
    1,
    0,
    0,
    0,
    993,
    157,
    1,
    0,
    0,
    0,
    994,
    998,
    3,
    162,
    81,
    0,
    995,
    998,
    3,
    188,
    94,
    0,
    996,
    998,
    3,
    160,
    80,
    0,
    997,
    994,
    1,
    0,
    0,
    0,
    997,
    995,
    1,
    0,
    0,
    0,
    997,
    996,
    1,
    0,
    0,
    0,
    998,
    159,
    1,
    0,
    0,
    0,
    999,
    1e3,
    3,
    194,
    97,
    0,
    1e3,
    1001,
    5,
    45,
    0,
    0,
    1001,
    1002,
    3,
    194,
    97,
    0,
    1002,
    161,
    1,
    0,
    0,
    0,
    1003,
    1008,
    5,
    59,
    0,
    0,
    1004,
    1008,
    5,
    44,
    0,
    0,
    1005,
    1008,
    5,
    66,
    0,
    0,
    1006,
    1008,
    3,
    194,
    97,
    0,
    1007,
    1003,
    1,
    0,
    0,
    0,
    1007,
    1004,
    1,
    0,
    0,
    0,
    1007,
    1005,
    1,
    0,
    0,
    0,
    1007,
    1006,
    1,
    0,
    0,
    0,
    1008,
    1021,
    1,
    0,
    0,
    0,
    1009,
    1011,
    5,
    23,
    0,
    0,
    1010,
    1012,
    3,
    158,
    79,
    0,
    1011,
    1010,
    1,
    0,
    0,
    0,
    1011,
    1012,
    1,
    0,
    0,
    0,
    1012,
    1017,
    1,
    0,
    0,
    0,
    1013,
    1014,
    5,
    16,
    0,
    0,
    1014,
    1016,
    3,
    158,
    79,
    0,
    1015,
    1013,
    1,
    0,
    0,
    0,
    1016,
    1019,
    1,
    0,
    0,
    0,
    1017,
    1015,
    1,
    0,
    0,
    0,
    1017,
    1018,
    1,
    0,
    0,
    0,
    1018,
    1020,
    1,
    0,
    0,
    0,
    1019,
    1017,
    1,
    0,
    0,
    0,
    1020,
    1022,
    5,
    24,
    0,
    0,
    1021,
    1009,
    1,
    0,
    0,
    0,
    1021,
    1022,
    1,
    0,
    0,
    0,
    1022,
    163,
    1,
    0,
    0,
    0,
    1023,
    1024,
    5,
    88,
    0,
    0,
    1024,
    1027,
    3,
    168,
    84,
    0,
    1025,
    1026,
    5,
    89,
    0,
    0,
    1026,
    1028,
    3,
    158,
    79,
    0,
    1027,
    1025,
    1,
    0,
    0,
    0,
    1027,
    1028,
    1,
    0,
    0,
    0,
    1028,
    165,
    1,
    0,
    0,
    0,
    1029,
    1030,
    3,
    168,
    84,
    0,
    1030,
    1031,
    5,
    89,
    0,
    0,
    1031,
    1032,
    3,
    158,
    79,
    0,
    1032,
    167,
    1,
    0,
    0,
    0,
    1033,
    1041,
    3,
    194,
    97,
    0,
    1034,
    1041,
    3,
    160,
    80,
    0,
    1035,
    1041,
    3,
    170,
    85,
    0,
    1036,
    1037,
    5,
    23,
    0,
    0,
    1037,
    1038,
    3,
    170,
    85,
    0,
    1038,
    1039,
    5,
    24,
    0,
    0,
    1039,
    1041,
    1,
    0,
    0,
    0,
    1040,
    1033,
    1,
    0,
    0,
    0,
    1040,
    1034,
    1,
    0,
    0,
    0,
    1040,
    1035,
    1,
    0,
    0,
    0,
    1040,
    1036,
    1,
    0,
    0,
    0,
    1041,
    169,
    1,
    0,
    0,
    0,
    1042,
    1047,
    3,
    194,
    97,
    0,
    1043,
    1044,
    5,
    16,
    0,
    0,
    1044,
    1046,
    3,
    194,
    97,
    0,
    1045,
    1043,
    1,
    0,
    0,
    0,
    1046,
    1049,
    1,
    0,
    0,
    0,
    1047,
    1045,
    1,
    0,
    0,
    0,
    1047,
    1048,
    1,
    0,
    0,
    0,
    1048,
    171,
    1,
    0,
    0,
    0,
    1049,
    1047,
    1,
    0,
    0,
    0,
    1050,
    1051,
    3,
    158,
    79,
    0,
    1051,
    1052,
    5,
    90,
    0,
    0,
    1052,
    1053,
    3,
    194,
    97,
    0,
    1053,
    173,
    1,
    0,
    0,
    0,
    1054,
    1055,
    3,
    194,
    97,
    0,
    1055,
    1056,
    5,
    70,
    0,
    0,
    1056,
    175,
    1,
    0,
    0,
    0,
    1057,
    1058,
    5,
    91,
    0,
    0,
    1058,
    1062,
    3,
    158,
    79,
    0,
    1059,
    1061,
    3,
    178,
    89,
    0,
    1060,
    1059,
    1,
    0,
    0,
    0,
    1061,
    1064,
    1,
    0,
    0,
    0,
    1062,
    1060,
    1,
    0,
    0,
    0,
    1062,
    1063,
    1,
    0,
    0,
    0,
    1063,
    177,
    1,
    0,
    0,
    0,
    1064,
    1062,
    1,
    0,
    0,
    0,
    1065,
    1066,
    5,
    92,
    0,
    0,
    1066,
    1067,
    3,
    188,
    94,
    0,
    1067,
    1068,
    3,
    154,
    77,
    0,
    1068,
    1072,
    1,
    0,
    0,
    0,
    1069,
    1070,
    5,
    93,
    0,
    0,
    1070,
    1072,
    3,
    154,
    77,
    0,
    1071,
    1065,
    1,
    0,
    0,
    0,
    1071,
    1069,
    1,
    0,
    0,
    0,
    1072,
    179,
    1,
    0,
    0,
    0,
    1073,
    1074,
    5,
    38,
    0,
    0,
    1074,
    1075,
    3,
    194,
    97,
    0,
    1075,
    1077,
    5,
    23,
    0,
    0,
    1076,
    1078,
    3,
    170,
    85,
    0,
    1077,
    1076,
    1,
    0,
    0,
    0,
    1077,
    1078,
    1,
    0,
    0,
    0,
    1078,
    1079,
    1,
    0,
    0,
    0,
    1079,
    1081,
    5,
    24,
    0,
    0,
    1080,
    1082,
    3,
    182,
    91,
    0,
    1081,
    1080,
    1,
    0,
    0,
    0,
    1081,
    1082,
    1,
    0,
    0,
    0,
    1082,
    1083,
    1,
    0,
    0,
    0,
    1083,
    1084,
    3,
    154,
    77,
    0,
    1084,
    181,
    1,
    0,
    0,
    0,
    1085,
    1086,
    5,
    94,
    0,
    0,
    1086,
    1087,
    3,
    170,
    85,
    0,
    1087,
    183,
    1,
    0,
    0,
    0,
    1088,
    1091,
    5,
    27,
    0,
    0,
    1089,
    1092,
    3,
    154,
    77,
    0,
    1090,
    1092,
    3,
    158,
    79,
    0,
    1091,
    1089,
    1,
    0,
    0,
    0,
    1091,
    1090,
    1,
    0,
    0,
    0,
    1092,
    1093,
    1,
    0,
    0,
    0,
    1093,
    1096,
    3,
    158,
    79,
    0,
    1094,
    1097,
    3,
    154,
    77,
    0,
    1095,
    1097,
    3,
    158,
    79,
    0,
    1096,
    1094,
    1,
    0,
    0,
    0,
    1096,
    1095,
    1,
    0,
    0,
    0,
    1097,
    1098,
    1,
    0,
    0,
    0,
    1098,
    1099,
    3,
    154,
    77,
    0,
    1099,
    185,
    1,
    0,
    0,
    0,
    1100,
    1101,
    5,
    51,
    0,
    0,
    1101,
    1102,
    3,
    158,
    79,
    0,
    1102,
    1103,
    3,
    154,
    77,
    0,
    1103,
    187,
    1,
    0,
    0,
    0,
    1104,
    1110,
    3,
    200,
    100,
    0,
    1105,
    1110,
    5,
    103,
    0,
    0,
    1106,
    1110,
    5,
    104,
    0,
    0,
    1107,
    1110,
    3,
    196,
    98,
    0,
    1108,
    1110,
    5,
    102,
    0,
    0,
    1109,
    1104,
    1,
    0,
    0,
    0,
    1109,
    1105,
    1,
    0,
    0,
    0,
    1109,
    1106,
    1,
    0,
    0,
    0,
    1109,
    1107,
    1,
    0,
    0,
    0,
    1109,
    1108,
    1,
    0,
    0,
    0,
    1110,
    189,
    1,
    0,
    0,
    0,
    1111,
    1113,
    5,
    23,
    0,
    0,
    1112,
    1114,
    3,
    140,
    70,
    0,
    1113,
    1112,
    1,
    0,
    0,
    0,
    1113,
    1114,
    1,
    0,
    0,
    0,
    1114,
    1121,
    1,
    0,
    0,
    0,
    1115,
    1117,
    5,
    16,
    0,
    0,
    1116,
    1118,
    3,
    140,
    70,
    0,
    1117,
    1116,
    1,
    0,
    0,
    0,
    1117,
    1118,
    1,
    0,
    0,
    0,
    1118,
    1120,
    1,
    0,
    0,
    0,
    1119,
    1115,
    1,
    0,
    0,
    0,
    1120,
    1123,
    1,
    0,
    0,
    0,
    1121,
    1119,
    1,
    0,
    0,
    0,
    1121,
    1122,
    1,
    0,
    0,
    0,
    1122,
    1124,
    1,
    0,
    0,
    0,
    1123,
    1121,
    1,
    0,
    0,
    0,
    1124,
    1138,
    5,
    24,
    0,
    0,
    1125,
    1134,
    5,
    42,
    0,
    0,
    1126,
    1131,
    3,
    140,
    70,
    0,
    1127,
    1128,
    5,
    16,
    0,
    0,
    1128,
    1130,
    3,
    140,
    70,
    0,
    1129,
    1127,
    1,
    0,
    0,
    0,
    1130,
    1133,
    1,
    0,
    0,
    0,
    1131,
    1129,
    1,
    0,
    0,
    0,
    1131,
    1132,
    1,
    0,
    0,
    0,
    1132,
    1135,
    1,
    0,
    0,
    0,
    1133,
    1131,
    1,
    0,
    0,
    0,
    1134,
    1126,
    1,
    0,
    0,
    0,
    1134,
    1135,
    1,
    0,
    0,
    0,
    1135,
    1136,
    1,
    0,
    0,
    0,
    1136,
    1138,
    5,
    43,
    0,
    0,
    1137,
    1111,
    1,
    0,
    0,
    0,
    1137,
    1125,
    1,
    0,
    0,
    0,
    1138,
    191,
    1,
    0,
    0,
    0,
    1139,
    1141,
    7,
    13,
    0,
    0,
    1140,
    1142,
    5,
    105,
    0,
    0,
    1141,
    1140,
    1,
    0,
    0,
    0,
    1141,
    1142,
    1,
    0,
    0,
    0,
    1142,
    193,
    1,
    0,
    0,
    0,
    1143,
    1144,
    7,
    14,
    0,
    0,
    1144,
    195,
    1,
    0,
    0,
    0,
    1145,
    1147,
    5,
    106,
    0,
    0,
    1146,
    1145,
    1,
    0,
    0,
    0,
    1147,
    1148,
    1,
    0,
    0,
    0,
    1148,
    1146,
    1,
    0,
    0,
    0,
    1148,
    1149,
    1,
    0,
    0,
    0,
    1149,
    197,
    1,
    0,
    0,
    0,
    1150,
    1162,
    5,
    96,
    0,
    0,
    1151,
    1152,
    5,
    23,
    0,
    0,
    1152,
    1157,
    3,
    78,
    39,
    0,
    1153,
    1154,
    5,
    16,
    0,
    0,
    1154,
    1156,
    3,
    78,
    39,
    0,
    1155,
    1153,
    1,
    0,
    0,
    0,
    1156,
    1159,
    1,
    0,
    0,
    0,
    1157,
    1155,
    1,
    0,
    0,
    0,
    1157,
    1158,
    1,
    0,
    0,
    0,
    1158,
    1160,
    1,
    0,
    0,
    0,
    1159,
    1157,
    1,
    0,
    0,
    0,
    1160,
    1161,
    5,
    24,
    0,
    0,
    1161,
    1163,
    1,
    0,
    0,
    0,
    1162,
    1151,
    1,
    0,
    0,
    0,
    1162,
    1163,
    1,
    0,
    0,
    0,
    1163,
    199,
    1,
    0,
    0,
    0,
    1164,
    1166,
    5,
    129,
    0,
    0,
    1165,
    1164,
    1,
    0,
    0,
    0,
    1166,
    1167,
    1,
    0,
    0,
    0,
    1167,
    1165,
    1,
    0,
    0,
    0,
    1167,
    1168,
    1,
    0,
    0,
    0,
    1168,
    201,
    1,
    0,
    0,
    0,
    130,
    213,
    215,
    230,
    234,
    239,
    245,
    249,
    252,
    257,
    263,
    270,
    274,
    287,
    295,
    300,
    310,
    313,
    319,
    327,
    330,
    341,
    350,
    352,
    358,
    385,
    388,
    399,
    404,
    409,
    423,
    426,
    433,
    437,
    439,
    444,
    449,
    452,
    458,
    462,
    466,
    471,
    484,
    486,
    493,
    503,
    509,
    520,
    523,
    529,
    532,
    540,
    543,
    549,
    552,
    560,
    563,
    569,
    573,
    584,
    589,
    594,
    602,
    607,
    613,
    618,
    631,
    633,
    638,
    648,
    668,
    680,
    685,
    691,
    695,
    698,
    710,
    719,
    723,
    726,
    733,
    739,
    761,
    783,
    787,
    792,
    796,
    800,
    805,
    810,
    814,
    838,
    892,
    896,
    912,
    914,
    926,
    933,
    941,
    945,
    953,
    957,
    959,
    970,
    992,
    997,
    1007,
    1011,
    1017,
    1021,
    1027,
    1040,
    1047,
    1062,
    1071,
    1077,
    1081,
    1091,
    1096,
    1109,
    1113,
    1117,
    1121,
    1131,
    1134,
    1137,
    1141,
    1148,
    1157,
    1162,
    1167
  ];
  SolidityParser.DecisionsToDFA = _SolidityParser._ATN.decisionToState.map((ds, index) => new u(ds, index));
  var SolidityParser_default = SolidityParser;
  var SourceUnitContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    EOF() {
      return this.getToken(SolidityParser.EOF, 0);
    }
    pragmaDirective_list() {
      return this.getTypedRuleContexts(PragmaDirectiveContext);
    }
    pragmaDirective(i2) {
      return this.getTypedRuleContext(PragmaDirectiveContext, i2);
    }
    importDirective_list() {
      return this.getTypedRuleContexts(ImportDirectiveContext);
    }
    importDirective(i2) {
      return this.getTypedRuleContext(ImportDirectiveContext, i2);
    }
    contractDefinition_list() {
      return this.getTypedRuleContexts(ContractDefinitionContext);
    }
    contractDefinition(i2) {
      return this.getTypedRuleContext(ContractDefinitionContext, i2);
    }
    enumDefinition_list() {
      return this.getTypedRuleContexts(EnumDefinitionContext);
    }
    enumDefinition(i2) {
      return this.getTypedRuleContext(EnumDefinitionContext, i2);
    }
    eventDefinition_list() {
      return this.getTypedRuleContexts(EventDefinitionContext);
    }
    eventDefinition(i2) {
      return this.getTypedRuleContext(EventDefinitionContext, i2);
    }
    structDefinition_list() {
      return this.getTypedRuleContexts(StructDefinitionContext);
    }
    structDefinition(i2) {
      return this.getTypedRuleContext(StructDefinitionContext, i2);
    }
    functionDefinition_list() {
      return this.getTypedRuleContexts(FunctionDefinitionContext);
    }
    functionDefinition(i2) {
      return this.getTypedRuleContext(FunctionDefinitionContext, i2);
    }
    fileLevelConstant_list() {
      return this.getTypedRuleContexts(FileLevelConstantContext);
    }
    fileLevelConstant(i2) {
      return this.getTypedRuleContext(FileLevelConstantContext, i2);
    }
    customErrorDefinition_list() {
      return this.getTypedRuleContexts(CustomErrorDefinitionContext);
    }
    customErrorDefinition(i2) {
      return this.getTypedRuleContext(CustomErrorDefinitionContext, i2);
    }
    typeDefinition_list() {
      return this.getTypedRuleContexts(TypeDefinitionContext);
    }
    typeDefinition(i2) {
      return this.getTypedRuleContext(TypeDefinitionContext, i2);
    }
    usingForDeclaration_list() {
      return this.getTypedRuleContexts(UsingForDeclarationContext);
    }
    usingForDeclaration(i2) {
      return this.getTypedRuleContext(UsingForDeclarationContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_sourceUnit;
    }
    enterRule(listener) {
      if (listener.enterSourceUnit) {
        listener.enterSourceUnit(this);
      }
    }
    exitRule(listener) {
      if (listener.exitSourceUnit) {
        listener.exitSourceUnit(this);
      }
    }
    accept(visitor) {
      if (visitor.visitSourceUnit) {
        return visitor.visitSourceUnit(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var PragmaDirectiveContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    pragmaName() {
      return this.getTypedRuleContext(PragmaNameContext, 0);
    }
    pragmaValue() {
      return this.getTypedRuleContext(PragmaValueContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_pragmaDirective;
    }
    enterRule(listener) {
      if (listener.enterPragmaDirective) {
        listener.enterPragmaDirective(this);
      }
    }
    exitRule(listener) {
      if (listener.exitPragmaDirective) {
        listener.exitPragmaDirective(this);
      }
    }
    accept(visitor) {
      if (visitor.visitPragmaDirective) {
        return visitor.visitPragmaDirective(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var PragmaNameContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_pragmaName;
    }
    enterRule(listener) {
      if (listener.enterPragmaName) {
        listener.enterPragmaName(this);
      }
    }
    exitRule(listener) {
      if (listener.exitPragmaName) {
        listener.exitPragmaName(this);
      }
    }
    accept(visitor) {
      if (visitor.visitPragmaName) {
        return visitor.visitPragmaName(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var PragmaValueContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    version() {
      return this.getTypedRuleContext(VersionContext, 0);
    }
    expression() {
      return this.getTypedRuleContext(ExpressionContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_pragmaValue;
    }
    enterRule(listener) {
      if (listener.enterPragmaValue) {
        listener.enterPragmaValue(this);
      }
    }
    exitRule(listener) {
      if (listener.exitPragmaValue) {
        listener.exitPragmaValue(this);
      }
    }
    accept(visitor) {
      if (visitor.visitPragmaValue) {
        return visitor.visitPragmaValue(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var VersionContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    versionConstraint_list() {
      return this.getTypedRuleContexts(VersionConstraintContext);
    }
    versionConstraint(i2) {
      return this.getTypedRuleContext(VersionConstraintContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_version;
    }
    enterRule(listener) {
      if (listener.enterVersion) {
        listener.enterVersion(this);
      }
    }
    exitRule(listener) {
      if (listener.exitVersion) {
        listener.exitVersion(this);
      }
    }
    accept(visitor) {
      if (visitor.visitVersion) {
        return visitor.visitVersion(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var VersionOperatorContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    get ruleIndex() {
      return SolidityParser.RULE_versionOperator;
    }
    enterRule(listener) {
      if (listener.enterVersionOperator) {
        listener.enterVersionOperator(this);
      }
    }
    exitRule(listener) {
      if (listener.exitVersionOperator) {
        listener.exitVersionOperator(this);
      }
    }
    accept(visitor) {
      if (visitor.visitVersionOperator) {
        return visitor.visitVersionOperator(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var VersionConstraintContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    VersionLiteral() {
      return this.getToken(SolidityParser.VersionLiteral, 0);
    }
    versionOperator() {
      return this.getTypedRuleContext(VersionOperatorContext, 0);
    }
    DecimalNumber() {
      return this.getToken(SolidityParser.DecimalNumber, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_versionConstraint;
    }
    enterRule(listener) {
      if (listener.enterVersionConstraint) {
        listener.enterVersionConstraint(this);
      }
    }
    exitRule(listener) {
      if (listener.exitVersionConstraint) {
        listener.exitVersionConstraint(this);
      }
    }
    accept(visitor) {
      if (visitor.visitVersionConstraint) {
        return visitor.visitVersionConstraint(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var ImportDeclarationContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifier_list() {
      return this.getTypedRuleContexts(IdentifierContext);
    }
    identifier(i2) {
      return this.getTypedRuleContext(IdentifierContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_importDeclaration;
    }
    enterRule(listener) {
      if (listener.enterImportDeclaration) {
        listener.enterImportDeclaration(this);
      }
    }
    exitRule(listener) {
      if (listener.exitImportDeclaration) {
        listener.exitImportDeclaration(this);
      }
    }
    accept(visitor) {
      if (visitor.visitImportDeclaration) {
        return visitor.visitImportDeclaration(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var ImportDirectiveContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    importPath() {
      return this.getTypedRuleContext(ImportPathContext, 0);
    }
    identifier_list() {
      return this.getTypedRuleContexts(IdentifierContext);
    }
    identifier(i2) {
      return this.getTypedRuleContext(IdentifierContext, i2);
    }
    importDeclaration_list() {
      return this.getTypedRuleContexts(ImportDeclarationContext);
    }
    importDeclaration(i2) {
      return this.getTypedRuleContext(ImportDeclarationContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_importDirective;
    }
    enterRule(listener) {
      if (listener.enterImportDirective) {
        listener.enterImportDirective(this);
      }
    }
    exitRule(listener) {
      if (listener.exitImportDirective) {
        listener.exitImportDirective(this);
      }
    }
    accept(visitor) {
      if (visitor.visitImportDirective) {
        return visitor.visitImportDirective(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var ImportPathContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    StringLiteralFragment() {
      return this.getToken(SolidityParser.StringLiteralFragment, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_importPath;
    }
    enterRule(listener) {
      if (listener.enterImportPath) {
        listener.enterImportPath(this);
      }
    }
    exitRule(listener) {
      if (listener.exitImportPath) {
        listener.exitImportPath(this);
      }
    }
    accept(visitor) {
      if (visitor.visitImportPath) {
        return visitor.visitImportPath(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var ContractDefinitionContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    inheritanceSpecifier_list() {
      return this.getTypedRuleContexts(InheritanceSpecifierContext);
    }
    inheritanceSpecifier(i2) {
      return this.getTypedRuleContext(InheritanceSpecifierContext, i2);
    }
    contractPart_list() {
      return this.getTypedRuleContexts(ContractPartContext);
    }
    contractPart(i2) {
      return this.getTypedRuleContext(ContractPartContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_contractDefinition;
    }
    enterRule(listener) {
      if (listener.enterContractDefinition) {
        listener.enterContractDefinition(this);
      }
    }
    exitRule(listener) {
      if (listener.exitContractDefinition) {
        listener.exitContractDefinition(this);
      }
    }
    accept(visitor) {
      if (visitor.visitContractDefinition) {
        return visitor.visitContractDefinition(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var InheritanceSpecifierContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    userDefinedTypeName() {
      return this.getTypedRuleContext(UserDefinedTypeNameContext, 0);
    }
    expressionList() {
      return this.getTypedRuleContext(ExpressionListContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_inheritanceSpecifier;
    }
    enterRule(listener) {
      if (listener.enterInheritanceSpecifier) {
        listener.enterInheritanceSpecifier(this);
      }
    }
    exitRule(listener) {
      if (listener.exitInheritanceSpecifier) {
        listener.exitInheritanceSpecifier(this);
      }
    }
    accept(visitor) {
      if (visitor.visitInheritanceSpecifier) {
        return visitor.visitInheritanceSpecifier(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var ContractPartContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    stateVariableDeclaration() {
      return this.getTypedRuleContext(StateVariableDeclarationContext, 0);
    }
    usingForDeclaration() {
      return this.getTypedRuleContext(UsingForDeclarationContext, 0);
    }
    structDefinition() {
      return this.getTypedRuleContext(StructDefinitionContext, 0);
    }
    modifierDefinition() {
      return this.getTypedRuleContext(ModifierDefinitionContext, 0);
    }
    functionDefinition() {
      return this.getTypedRuleContext(FunctionDefinitionContext, 0);
    }
    eventDefinition() {
      return this.getTypedRuleContext(EventDefinitionContext, 0);
    }
    enumDefinition() {
      return this.getTypedRuleContext(EnumDefinitionContext, 0);
    }
    customErrorDefinition() {
      return this.getTypedRuleContext(CustomErrorDefinitionContext, 0);
    }
    typeDefinition() {
      return this.getTypedRuleContext(TypeDefinitionContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_contractPart;
    }
    enterRule(listener) {
      if (listener.enterContractPart) {
        listener.enterContractPart(this);
      }
    }
    exitRule(listener) {
      if (listener.exitContractPart) {
        listener.exitContractPart(this);
      }
    }
    accept(visitor) {
      if (visitor.visitContractPart) {
        return visitor.visitContractPart(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var StateVariableDeclarationContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    typeName() {
      return this.getTypedRuleContext(TypeNameContext, 0);
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    PublicKeyword_list() {
      return this.getTokens(SolidityParser.PublicKeyword);
    }
    PublicKeyword(i2) {
      return this.getToken(SolidityParser.PublicKeyword, i2);
    }
    InternalKeyword_list() {
      return this.getTokens(SolidityParser.InternalKeyword);
    }
    InternalKeyword(i2) {
      return this.getToken(SolidityParser.InternalKeyword, i2);
    }
    PrivateKeyword_list() {
      return this.getTokens(SolidityParser.PrivateKeyword);
    }
    PrivateKeyword(i2) {
      return this.getToken(SolidityParser.PrivateKeyword, i2);
    }
    ConstantKeyword_list() {
      return this.getTokens(SolidityParser.ConstantKeyword);
    }
    ConstantKeyword(i2) {
      return this.getToken(SolidityParser.ConstantKeyword, i2);
    }
    ImmutableKeyword_list() {
      return this.getTokens(SolidityParser.ImmutableKeyword);
    }
    ImmutableKeyword(i2) {
      return this.getToken(SolidityParser.ImmutableKeyword, i2);
    }
    overrideSpecifier_list() {
      return this.getTypedRuleContexts(OverrideSpecifierContext);
    }
    overrideSpecifier(i2) {
      return this.getTypedRuleContext(OverrideSpecifierContext, i2);
    }
    expression() {
      return this.getTypedRuleContext(ExpressionContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_stateVariableDeclaration;
    }
    enterRule(listener) {
      if (listener.enterStateVariableDeclaration) {
        listener.enterStateVariableDeclaration(this);
      }
    }
    exitRule(listener) {
      if (listener.exitStateVariableDeclaration) {
        listener.exitStateVariableDeclaration(this);
      }
    }
    accept(visitor) {
      if (visitor.visitStateVariableDeclaration) {
        return visitor.visitStateVariableDeclaration(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var FileLevelConstantContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    typeName() {
      return this.getTypedRuleContext(TypeNameContext, 0);
    }
    ConstantKeyword() {
      return this.getToken(SolidityParser.ConstantKeyword, 0);
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    expression() {
      return this.getTypedRuleContext(ExpressionContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_fileLevelConstant;
    }
    enterRule(listener) {
      if (listener.enterFileLevelConstant) {
        listener.enterFileLevelConstant(this);
      }
    }
    exitRule(listener) {
      if (listener.exitFileLevelConstant) {
        listener.exitFileLevelConstant(this);
      }
    }
    accept(visitor) {
      if (visitor.visitFileLevelConstant) {
        return visitor.visitFileLevelConstant(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var CustomErrorDefinitionContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    parameterList() {
      return this.getTypedRuleContext(ParameterListContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_customErrorDefinition;
    }
    enterRule(listener) {
      if (listener.enterCustomErrorDefinition) {
        listener.enterCustomErrorDefinition(this);
      }
    }
    exitRule(listener) {
      if (listener.exitCustomErrorDefinition) {
        listener.exitCustomErrorDefinition(this);
      }
    }
    accept(visitor) {
      if (visitor.visitCustomErrorDefinition) {
        return visitor.visitCustomErrorDefinition(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var TypeDefinitionContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    TypeKeyword() {
      return this.getToken(SolidityParser.TypeKeyword, 0);
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    elementaryTypeName() {
      return this.getTypedRuleContext(ElementaryTypeNameContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_typeDefinition;
    }
    enterRule(listener) {
      if (listener.enterTypeDefinition) {
        listener.enterTypeDefinition(this);
      }
    }
    exitRule(listener) {
      if (listener.exitTypeDefinition) {
        listener.exitTypeDefinition(this);
      }
    }
    accept(visitor) {
      if (visitor.visitTypeDefinition) {
        return visitor.visitTypeDefinition(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var UsingForDeclarationContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    usingForObject() {
      return this.getTypedRuleContext(UsingForObjectContext, 0);
    }
    typeName() {
      return this.getTypedRuleContext(TypeNameContext, 0);
    }
    GlobalKeyword() {
      return this.getToken(SolidityParser.GlobalKeyword, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_usingForDeclaration;
    }
    enterRule(listener) {
      if (listener.enterUsingForDeclaration) {
        listener.enterUsingForDeclaration(this);
      }
    }
    exitRule(listener) {
      if (listener.exitUsingForDeclaration) {
        listener.exitUsingForDeclaration(this);
      }
    }
    accept(visitor) {
      if (visitor.visitUsingForDeclaration) {
        return visitor.visitUsingForDeclaration(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var UsingForObjectContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    userDefinedTypeName() {
      return this.getTypedRuleContext(UserDefinedTypeNameContext, 0);
    }
    usingForObjectDirective_list() {
      return this.getTypedRuleContexts(UsingForObjectDirectiveContext);
    }
    usingForObjectDirective(i2) {
      return this.getTypedRuleContext(UsingForObjectDirectiveContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_usingForObject;
    }
    enterRule(listener) {
      if (listener.enterUsingForObject) {
        listener.enterUsingForObject(this);
      }
    }
    exitRule(listener) {
      if (listener.exitUsingForObject) {
        listener.exitUsingForObject(this);
      }
    }
    accept(visitor) {
      if (visitor.visitUsingForObject) {
        return visitor.visitUsingForObject(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var UsingForObjectDirectiveContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    userDefinedTypeName() {
      return this.getTypedRuleContext(UserDefinedTypeNameContext, 0);
    }
    userDefinableOperators() {
      return this.getTypedRuleContext(UserDefinableOperatorsContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_usingForObjectDirective;
    }
    enterRule(listener) {
      if (listener.enterUsingForObjectDirective) {
        listener.enterUsingForObjectDirective(this);
      }
    }
    exitRule(listener) {
      if (listener.exitUsingForObjectDirective) {
        listener.exitUsingForObjectDirective(this);
      }
    }
    accept(visitor) {
      if (visitor.visitUsingForObjectDirective) {
        return visitor.visitUsingForObjectDirective(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var UserDefinableOperatorsContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    get ruleIndex() {
      return SolidityParser.RULE_userDefinableOperators;
    }
    enterRule(listener) {
      if (listener.enterUserDefinableOperators) {
        listener.enterUserDefinableOperators(this);
      }
    }
    exitRule(listener) {
      if (listener.exitUserDefinableOperators) {
        listener.exitUserDefinableOperators(this);
      }
    }
    accept(visitor) {
      if (visitor.visitUserDefinableOperators) {
        return visitor.visitUserDefinableOperators(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var StructDefinitionContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    variableDeclaration_list() {
      return this.getTypedRuleContexts(VariableDeclarationContext);
    }
    variableDeclaration(i2) {
      return this.getTypedRuleContext(VariableDeclarationContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_structDefinition;
    }
    enterRule(listener) {
      if (listener.enterStructDefinition) {
        listener.enterStructDefinition(this);
      }
    }
    exitRule(listener) {
      if (listener.exitStructDefinition) {
        listener.exitStructDefinition(this);
      }
    }
    accept(visitor) {
      if (visitor.visitStructDefinition) {
        return visitor.visitStructDefinition(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var ModifierDefinitionContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    block() {
      return this.getTypedRuleContext(BlockContext, 0);
    }
    parameterList() {
      return this.getTypedRuleContext(ParameterListContext, 0);
    }
    VirtualKeyword_list() {
      return this.getTokens(SolidityParser.VirtualKeyword);
    }
    VirtualKeyword(i2) {
      return this.getToken(SolidityParser.VirtualKeyword, i2);
    }
    overrideSpecifier_list() {
      return this.getTypedRuleContexts(OverrideSpecifierContext);
    }
    overrideSpecifier(i2) {
      return this.getTypedRuleContext(OverrideSpecifierContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_modifierDefinition;
    }
    enterRule(listener) {
      if (listener.enterModifierDefinition) {
        listener.enterModifierDefinition(this);
      }
    }
    exitRule(listener) {
      if (listener.exitModifierDefinition) {
        listener.exitModifierDefinition(this);
      }
    }
    accept(visitor) {
      if (visitor.visitModifierDefinition) {
        return visitor.visitModifierDefinition(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var ModifierInvocationContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    expressionList() {
      return this.getTypedRuleContext(ExpressionListContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_modifierInvocation;
    }
    enterRule(listener) {
      if (listener.enterModifierInvocation) {
        listener.enterModifierInvocation(this);
      }
    }
    exitRule(listener) {
      if (listener.exitModifierInvocation) {
        listener.exitModifierInvocation(this);
      }
    }
    accept(visitor) {
      if (visitor.visitModifierInvocation) {
        return visitor.visitModifierInvocation(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var FunctionDefinitionContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    functionDescriptor() {
      return this.getTypedRuleContext(FunctionDescriptorContext, 0);
    }
    parameterList() {
      return this.getTypedRuleContext(ParameterListContext, 0);
    }
    modifierList() {
      return this.getTypedRuleContext(ModifierListContext, 0);
    }
    block() {
      return this.getTypedRuleContext(BlockContext, 0);
    }
    returnParameters() {
      return this.getTypedRuleContext(ReturnParametersContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_functionDefinition;
    }
    enterRule(listener) {
      if (listener.enterFunctionDefinition) {
        listener.enterFunctionDefinition(this);
      }
    }
    exitRule(listener) {
      if (listener.exitFunctionDefinition) {
        listener.exitFunctionDefinition(this);
      }
    }
    accept(visitor) {
      if (visitor.visitFunctionDefinition) {
        return visitor.visitFunctionDefinition(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var FunctionDescriptorContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    ConstructorKeyword() {
      return this.getToken(SolidityParser.ConstructorKeyword, 0);
    }
    FallbackKeyword() {
      return this.getToken(SolidityParser.FallbackKeyword, 0);
    }
    ReceiveKeyword() {
      return this.getToken(SolidityParser.ReceiveKeyword, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_functionDescriptor;
    }
    enterRule(listener) {
      if (listener.enterFunctionDescriptor) {
        listener.enterFunctionDescriptor(this);
      }
    }
    exitRule(listener) {
      if (listener.exitFunctionDescriptor) {
        listener.exitFunctionDescriptor(this);
      }
    }
    accept(visitor) {
      if (visitor.visitFunctionDescriptor) {
        return visitor.visitFunctionDescriptor(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var ReturnParametersContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    parameterList() {
      return this.getTypedRuleContext(ParameterListContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_returnParameters;
    }
    enterRule(listener) {
      if (listener.enterReturnParameters) {
        listener.enterReturnParameters(this);
      }
    }
    exitRule(listener) {
      if (listener.exitReturnParameters) {
        listener.exitReturnParameters(this);
      }
    }
    accept(visitor) {
      if (visitor.visitReturnParameters) {
        return visitor.visitReturnParameters(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var ModifierListContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    ExternalKeyword_list() {
      return this.getTokens(SolidityParser.ExternalKeyword);
    }
    ExternalKeyword(i2) {
      return this.getToken(SolidityParser.ExternalKeyword, i2);
    }
    PublicKeyword_list() {
      return this.getTokens(SolidityParser.PublicKeyword);
    }
    PublicKeyword(i2) {
      return this.getToken(SolidityParser.PublicKeyword, i2);
    }
    InternalKeyword_list() {
      return this.getTokens(SolidityParser.InternalKeyword);
    }
    InternalKeyword(i2) {
      return this.getToken(SolidityParser.InternalKeyword, i2);
    }
    PrivateKeyword_list() {
      return this.getTokens(SolidityParser.PrivateKeyword);
    }
    PrivateKeyword(i2) {
      return this.getToken(SolidityParser.PrivateKeyword, i2);
    }
    VirtualKeyword_list() {
      return this.getTokens(SolidityParser.VirtualKeyword);
    }
    VirtualKeyword(i2) {
      return this.getToken(SolidityParser.VirtualKeyword, i2);
    }
    stateMutability_list() {
      return this.getTypedRuleContexts(StateMutabilityContext);
    }
    stateMutability(i2) {
      return this.getTypedRuleContext(StateMutabilityContext, i2);
    }
    modifierInvocation_list() {
      return this.getTypedRuleContexts(ModifierInvocationContext);
    }
    modifierInvocation(i2) {
      return this.getTypedRuleContext(ModifierInvocationContext, i2);
    }
    overrideSpecifier_list() {
      return this.getTypedRuleContexts(OverrideSpecifierContext);
    }
    overrideSpecifier(i2) {
      return this.getTypedRuleContext(OverrideSpecifierContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_modifierList;
    }
    enterRule(listener) {
      if (listener.enterModifierList) {
        listener.enterModifierList(this);
      }
    }
    exitRule(listener) {
      if (listener.exitModifierList) {
        listener.exitModifierList(this);
      }
    }
    accept(visitor) {
      if (visitor.visitModifierList) {
        return visitor.visitModifierList(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var EventDefinitionContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    eventParameterList() {
      return this.getTypedRuleContext(EventParameterListContext, 0);
    }
    AnonymousKeyword() {
      return this.getToken(SolidityParser.AnonymousKeyword, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_eventDefinition;
    }
    enterRule(listener) {
      if (listener.enterEventDefinition) {
        listener.enterEventDefinition(this);
      }
    }
    exitRule(listener) {
      if (listener.exitEventDefinition) {
        listener.exitEventDefinition(this);
      }
    }
    accept(visitor) {
      if (visitor.visitEventDefinition) {
        return visitor.visitEventDefinition(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var EnumValueContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_enumValue;
    }
    enterRule(listener) {
      if (listener.enterEnumValue) {
        listener.enterEnumValue(this);
      }
    }
    exitRule(listener) {
      if (listener.exitEnumValue) {
        listener.exitEnumValue(this);
      }
    }
    accept(visitor) {
      if (visitor.visitEnumValue) {
        return visitor.visitEnumValue(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var EnumDefinitionContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    enumValue_list() {
      return this.getTypedRuleContexts(EnumValueContext);
    }
    enumValue(i2) {
      return this.getTypedRuleContext(EnumValueContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_enumDefinition;
    }
    enterRule(listener) {
      if (listener.enterEnumDefinition) {
        listener.enterEnumDefinition(this);
      }
    }
    exitRule(listener) {
      if (listener.exitEnumDefinition) {
        listener.exitEnumDefinition(this);
      }
    }
    accept(visitor) {
      if (visitor.visitEnumDefinition) {
        return visitor.visitEnumDefinition(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var ParameterListContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    parameter_list() {
      return this.getTypedRuleContexts(ParameterContext);
    }
    parameter(i2) {
      return this.getTypedRuleContext(ParameterContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_parameterList;
    }
    enterRule(listener) {
      if (listener.enterParameterList) {
        listener.enterParameterList(this);
      }
    }
    exitRule(listener) {
      if (listener.exitParameterList) {
        listener.exitParameterList(this);
      }
    }
    accept(visitor) {
      if (visitor.visitParameterList) {
        return visitor.visitParameterList(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var ParameterContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    typeName() {
      return this.getTypedRuleContext(TypeNameContext, 0);
    }
    storageLocation() {
      return this.getTypedRuleContext(StorageLocationContext, 0);
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_parameter;
    }
    enterRule(listener) {
      if (listener.enterParameter) {
        listener.enterParameter(this);
      }
    }
    exitRule(listener) {
      if (listener.exitParameter) {
        listener.exitParameter(this);
      }
    }
    accept(visitor) {
      if (visitor.visitParameter) {
        return visitor.visitParameter(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var EventParameterListContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    eventParameter_list() {
      return this.getTypedRuleContexts(EventParameterContext);
    }
    eventParameter(i2) {
      return this.getTypedRuleContext(EventParameterContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_eventParameterList;
    }
    enterRule(listener) {
      if (listener.enterEventParameterList) {
        listener.enterEventParameterList(this);
      }
    }
    exitRule(listener) {
      if (listener.exitEventParameterList) {
        listener.exitEventParameterList(this);
      }
    }
    accept(visitor) {
      if (visitor.visitEventParameterList) {
        return visitor.visitEventParameterList(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var EventParameterContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    typeName() {
      return this.getTypedRuleContext(TypeNameContext, 0);
    }
    IndexedKeyword() {
      return this.getToken(SolidityParser.IndexedKeyword, 0);
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_eventParameter;
    }
    enterRule(listener) {
      if (listener.enterEventParameter) {
        listener.enterEventParameter(this);
      }
    }
    exitRule(listener) {
      if (listener.exitEventParameter) {
        listener.exitEventParameter(this);
      }
    }
    accept(visitor) {
      if (visitor.visitEventParameter) {
        return visitor.visitEventParameter(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var FunctionTypeParameterListContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    functionTypeParameter_list() {
      return this.getTypedRuleContexts(FunctionTypeParameterContext);
    }
    functionTypeParameter(i2) {
      return this.getTypedRuleContext(FunctionTypeParameterContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_functionTypeParameterList;
    }
    enterRule(listener) {
      if (listener.enterFunctionTypeParameterList) {
        listener.enterFunctionTypeParameterList(this);
      }
    }
    exitRule(listener) {
      if (listener.exitFunctionTypeParameterList) {
        listener.exitFunctionTypeParameterList(this);
      }
    }
    accept(visitor) {
      if (visitor.visitFunctionTypeParameterList) {
        return visitor.visitFunctionTypeParameterList(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var FunctionTypeParameterContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    typeName() {
      return this.getTypedRuleContext(TypeNameContext, 0);
    }
    storageLocation() {
      return this.getTypedRuleContext(StorageLocationContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_functionTypeParameter;
    }
    enterRule(listener) {
      if (listener.enterFunctionTypeParameter) {
        listener.enterFunctionTypeParameter(this);
      }
    }
    exitRule(listener) {
      if (listener.exitFunctionTypeParameter) {
        listener.exitFunctionTypeParameter(this);
      }
    }
    accept(visitor) {
      if (visitor.visitFunctionTypeParameter) {
        return visitor.visitFunctionTypeParameter(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var VariableDeclarationContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    typeName() {
      return this.getTypedRuleContext(TypeNameContext, 0);
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    storageLocation() {
      return this.getTypedRuleContext(StorageLocationContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_variableDeclaration;
    }
    enterRule(listener) {
      if (listener.enterVariableDeclaration) {
        listener.enterVariableDeclaration(this);
      }
    }
    exitRule(listener) {
      if (listener.exitVariableDeclaration) {
        listener.exitVariableDeclaration(this);
      }
    }
    accept(visitor) {
      if (visitor.visitVariableDeclaration) {
        return visitor.visitVariableDeclaration(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var TypeNameContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    elementaryTypeName() {
      return this.getTypedRuleContext(ElementaryTypeNameContext, 0);
    }
    userDefinedTypeName() {
      return this.getTypedRuleContext(UserDefinedTypeNameContext, 0);
    }
    mapping() {
      return this.getTypedRuleContext(MappingContext, 0);
    }
    functionTypeName() {
      return this.getTypedRuleContext(FunctionTypeNameContext, 0);
    }
    PayableKeyword() {
      return this.getToken(SolidityParser.PayableKeyword, 0);
    }
    typeName() {
      return this.getTypedRuleContext(TypeNameContext, 0);
    }
    expression() {
      return this.getTypedRuleContext(ExpressionContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_typeName;
    }
    enterRule(listener) {
      if (listener.enterTypeName) {
        listener.enterTypeName(this);
      }
    }
    exitRule(listener) {
      if (listener.exitTypeName) {
        listener.exitTypeName(this);
      }
    }
    accept(visitor) {
      if (visitor.visitTypeName) {
        return visitor.visitTypeName(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var UserDefinedTypeNameContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifier_list() {
      return this.getTypedRuleContexts(IdentifierContext);
    }
    identifier(i2) {
      return this.getTypedRuleContext(IdentifierContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_userDefinedTypeName;
    }
    enterRule(listener) {
      if (listener.enterUserDefinedTypeName) {
        listener.enterUserDefinedTypeName(this);
      }
    }
    exitRule(listener) {
      if (listener.exitUserDefinedTypeName) {
        listener.exitUserDefinedTypeName(this);
      }
    }
    accept(visitor) {
      if (visitor.visitUserDefinedTypeName) {
        return visitor.visitUserDefinedTypeName(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var MappingKeyContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    elementaryTypeName() {
      return this.getTypedRuleContext(ElementaryTypeNameContext, 0);
    }
    userDefinedTypeName() {
      return this.getTypedRuleContext(UserDefinedTypeNameContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_mappingKey;
    }
    enterRule(listener) {
      if (listener.enterMappingKey) {
        listener.enterMappingKey(this);
      }
    }
    exitRule(listener) {
      if (listener.exitMappingKey) {
        listener.exitMappingKey(this);
      }
    }
    accept(visitor) {
      if (visitor.visitMappingKey) {
        return visitor.visitMappingKey(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var MappingContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    mappingKey() {
      return this.getTypedRuleContext(MappingKeyContext, 0);
    }
    typeName() {
      return this.getTypedRuleContext(TypeNameContext, 0);
    }
    mappingKeyName() {
      return this.getTypedRuleContext(MappingKeyNameContext, 0);
    }
    mappingValueName() {
      return this.getTypedRuleContext(MappingValueNameContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_mapping;
    }
    enterRule(listener) {
      if (listener.enterMapping) {
        listener.enterMapping(this);
      }
    }
    exitRule(listener) {
      if (listener.exitMapping) {
        listener.exitMapping(this);
      }
    }
    accept(visitor) {
      if (visitor.visitMapping) {
        return visitor.visitMapping(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var MappingKeyNameContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_mappingKeyName;
    }
    enterRule(listener) {
      if (listener.enterMappingKeyName) {
        listener.enterMappingKeyName(this);
      }
    }
    exitRule(listener) {
      if (listener.exitMappingKeyName) {
        listener.exitMappingKeyName(this);
      }
    }
    accept(visitor) {
      if (visitor.visitMappingKeyName) {
        return visitor.visitMappingKeyName(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var MappingValueNameContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_mappingValueName;
    }
    enterRule(listener) {
      if (listener.enterMappingValueName) {
        listener.enterMappingValueName(this);
      }
    }
    exitRule(listener) {
      if (listener.exitMappingValueName) {
        listener.exitMappingValueName(this);
      }
    }
    accept(visitor) {
      if (visitor.visitMappingValueName) {
        return visitor.visitMappingValueName(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var FunctionTypeNameContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    functionTypeParameterList_list() {
      return this.getTypedRuleContexts(FunctionTypeParameterListContext);
    }
    functionTypeParameterList(i2) {
      return this.getTypedRuleContext(FunctionTypeParameterListContext, i2);
    }
    InternalKeyword_list() {
      return this.getTokens(SolidityParser.InternalKeyword);
    }
    InternalKeyword(i2) {
      return this.getToken(SolidityParser.InternalKeyword, i2);
    }
    ExternalKeyword_list() {
      return this.getTokens(SolidityParser.ExternalKeyword);
    }
    ExternalKeyword(i2) {
      return this.getToken(SolidityParser.ExternalKeyword, i2);
    }
    stateMutability_list() {
      return this.getTypedRuleContexts(StateMutabilityContext);
    }
    stateMutability(i2) {
      return this.getTypedRuleContext(StateMutabilityContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_functionTypeName;
    }
    enterRule(listener) {
      if (listener.enterFunctionTypeName) {
        listener.enterFunctionTypeName(this);
      }
    }
    exitRule(listener) {
      if (listener.exitFunctionTypeName) {
        listener.exitFunctionTypeName(this);
      }
    }
    accept(visitor) {
      if (visitor.visitFunctionTypeName) {
        return visitor.visitFunctionTypeName(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var StorageLocationContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    get ruleIndex() {
      return SolidityParser.RULE_storageLocation;
    }
    enterRule(listener) {
      if (listener.enterStorageLocation) {
        listener.enterStorageLocation(this);
      }
    }
    exitRule(listener) {
      if (listener.exitStorageLocation) {
        listener.exitStorageLocation(this);
      }
    }
    accept(visitor) {
      if (visitor.visitStorageLocation) {
        return visitor.visitStorageLocation(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var StateMutabilityContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    PureKeyword() {
      return this.getToken(SolidityParser.PureKeyword, 0);
    }
    ConstantKeyword() {
      return this.getToken(SolidityParser.ConstantKeyword, 0);
    }
    ViewKeyword() {
      return this.getToken(SolidityParser.ViewKeyword, 0);
    }
    PayableKeyword() {
      return this.getToken(SolidityParser.PayableKeyword, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_stateMutability;
    }
    enterRule(listener) {
      if (listener.enterStateMutability) {
        listener.enterStateMutability(this);
      }
    }
    exitRule(listener) {
      if (listener.exitStateMutability) {
        listener.exitStateMutability(this);
      }
    }
    accept(visitor) {
      if (visitor.visitStateMutability) {
        return visitor.visitStateMutability(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var BlockContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    statement_list() {
      return this.getTypedRuleContexts(StatementContext);
    }
    statement(i2) {
      return this.getTypedRuleContext(StatementContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_block;
    }
    enterRule(listener) {
      if (listener.enterBlock) {
        listener.enterBlock(this);
      }
    }
    exitRule(listener) {
      if (listener.exitBlock) {
        listener.exitBlock(this);
      }
    }
    accept(visitor) {
      if (visitor.visitBlock) {
        return visitor.visitBlock(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var StatementContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    ifStatement() {
      return this.getTypedRuleContext(IfStatementContext, 0);
    }
    tryStatement() {
      return this.getTypedRuleContext(TryStatementContext, 0);
    }
    whileStatement() {
      return this.getTypedRuleContext(WhileStatementContext, 0);
    }
    forStatement() {
      return this.getTypedRuleContext(ForStatementContext, 0);
    }
    block() {
      return this.getTypedRuleContext(BlockContext, 0);
    }
    inlineAssemblyStatement() {
      return this.getTypedRuleContext(InlineAssemblyStatementContext, 0);
    }
    doWhileStatement() {
      return this.getTypedRuleContext(DoWhileStatementContext, 0);
    }
    continueStatement() {
      return this.getTypedRuleContext(ContinueStatementContext, 0);
    }
    breakStatement() {
      return this.getTypedRuleContext(BreakStatementContext, 0);
    }
    returnStatement() {
      return this.getTypedRuleContext(ReturnStatementContext, 0);
    }
    throwStatement() {
      return this.getTypedRuleContext(ThrowStatementContext, 0);
    }
    emitStatement() {
      return this.getTypedRuleContext(EmitStatementContext, 0);
    }
    simpleStatement() {
      return this.getTypedRuleContext(SimpleStatementContext, 0);
    }
    uncheckedStatement() {
      return this.getTypedRuleContext(UncheckedStatementContext, 0);
    }
    revertStatement() {
      return this.getTypedRuleContext(RevertStatementContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_statement;
    }
    enterRule(listener) {
      if (listener.enterStatement) {
        listener.enterStatement(this);
      }
    }
    exitRule(listener) {
      if (listener.exitStatement) {
        listener.exitStatement(this);
      }
    }
    accept(visitor) {
      if (visitor.visitStatement) {
        return visitor.visitStatement(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var ExpressionStatementContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    expression() {
      return this.getTypedRuleContext(ExpressionContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_expressionStatement;
    }
    enterRule(listener) {
      if (listener.enterExpressionStatement) {
        listener.enterExpressionStatement(this);
      }
    }
    exitRule(listener) {
      if (listener.exitExpressionStatement) {
        listener.exitExpressionStatement(this);
      }
    }
    accept(visitor) {
      if (visitor.visitExpressionStatement) {
        return visitor.visitExpressionStatement(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var IfStatementContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    expression() {
      return this.getTypedRuleContext(ExpressionContext, 0);
    }
    statement_list() {
      return this.getTypedRuleContexts(StatementContext);
    }
    statement(i2) {
      return this.getTypedRuleContext(StatementContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_ifStatement;
    }
    enterRule(listener) {
      if (listener.enterIfStatement) {
        listener.enterIfStatement(this);
      }
    }
    exitRule(listener) {
      if (listener.exitIfStatement) {
        listener.exitIfStatement(this);
      }
    }
    accept(visitor) {
      if (visitor.visitIfStatement) {
        return visitor.visitIfStatement(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var TryStatementContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    expression() {
      return this.getTypedRuleContext(ExpressionContext, 0);
    }
    block() {
      return this.getTypedRuleContext(BlockContext, 0);
    }
    returnParameters() {
      return this.getTypedRuleContext(ReturnParametersContext, 0);
    }
    catchClause_list() {
      return this.getTypedRuleContexts(CatchClauseContext);
    }
    catchClause(i2) {
      return this.getTypedRuleContext(CatchClauseContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_tryStatement;
    }
    enterRule(listener) {
      if (listener.enterTryStatement) {
        listener.enterTryStatement(this);
      }
    }
    exitRule(listener) {
      if (listener.exitTryStatement) {
        listener.exitTryStatement(this);
      }
    }
    accept(visitor) {
      if (visitor.visitTryStatement) {
        return visitor.visitTryStatement(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var CatchClauseContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    block() {
      return this.getTypedRuleContext(BlockContext, 0);
    }
    parameterList() {
      return this.getTypedRuleContext(ParameterListContext, 0);
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_catchClause;
    }
    enterRule(listener) {
      if (listener.enterCatchClause) {
        listener.enterCatchClause(this);
      }
    }
    exitRule(listener) {
      if (listener.exitCatchClause) {
        listener.exitCatchClause(this);
      }
    }
    accept(visitor) {
      if (visitor.visitCatchClause) {
        return visitor.visitCatchClause(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var WhileStatementContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    expression() {
      return this.getTypedRuleContext(ExpressionContext, 0);
    }
    statement() {
      return this.getTypedRuleContext(StatementContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_whileStatement;
    }
    enterRule(listener) {
      if (listener.enterWhileStatement) {
        listener.enterWhileStatement(this);
      }
    }
    exitRule(listener) {
      if (listener.exitWhileStatement) {
        listener.exitWhileStatement(this);
      }
    }
    accept(visitor) {
      if (visitor.visitWhileStatement) {
        return visitor.visitWhileStatement(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var SimpleStatementContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    variableDeclarationStatement() {
      return this.getTypedRuleContext(VariableDeclarationStatementContext, 0);
    }
    expressionStatement() {
      return this.getTypedRuleContext(ExpressionStatementContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_simpleStatement;
    }
    enterRule(listener) {
      if (listener.enterSimpleStatement) {
        listener.enterSimpleStatement(this);
      }
    }
    exitRule(listener) {
      if (listener.exitSimpleStatement) {
        listener.exitSimpleStatement(this);
      }
    }
    accept(visitor) {
      if (visitor.visitSimpleStatement) {
        return visitor.visitSimpleStatement(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var UncheckedStatementContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    block() {
      return this.getTypedRuleContext(BlockContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_uncheckedStatement;
    }
    enterRule(listener) {
      if (listener.enterUncheckedStatement) {
        listener.enterUncheckedStatement(this);
      }
    }
    exitRule(listener) {
      if (listener.exitUncheckedStatement) {
        listener.exitUncheckedStatement(this);
      }
    }
    accept(visitor) {
      if (visitor.visitUncheckedStatement) {
        return visitor.visitUncheckedStatement(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var ForStatementContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    statement() {
      return this.getTypedRuleContext(StatementContext, 0);
    }
    simpleStatement() {
      return this.getTypedRuleContext(SimpleStatementContext, 0);
    }
    expressionStatement() {
      return this.getTypedRuleContext(ExpressionStatementContext, 0);
    }
    expression() {
      return this.getTypedRuleContext(ExpressionContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_forStatement;
    }
    enterRule(listener) {
      if (listener.enterForStatement) {
        listener.enterForStatement(this);
      }
    }
    exitRule(listener) {
      if (listener.exitForStatement) {
        listener.exitForStatement(this);
      }
    }
    accept(visitor) {
      if (visitor.visitForStatement) {
        return visitor.visitForStatement(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var InlineAssemblyStatementContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    assemblyBlock() {
      return this.getTypedRuleContext(AssemblyBlockContext, 0);
    }
    StringLiteralFragment() {
      return this.getToken(SolidityParser.StringLiteralFragment, 0);
    }
    inlineAssemblyStatementFlag() {
      return this.getTypedRuleContext(InlineAssemblyStatementFlagContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_inlineAssemblyStatement;
    }
    enterRule(listener) {
      if (listener.enterInlineAssemblyStatement) {
        listener.enterInlineAssemblyStatement(this);
      }
    }
    exitRule(listener) {
      if (listener.exitInlineAssemblyStatement) {
        listener.exitInlineAssemblyStatement(this);
      }
    }
    accept(visitor) {
      if (visitor.visitInlineAssemblyStatement) {
        return visitor.visitInlineAssemblyStatement(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var InlineAssemblyStatementFlagContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    stringLiteral() {
      return this.getTypedRuleContext(StringLiteralContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_inlineAssemblyStatementFlag;
    }
    enterRule(listener) {
      if (listener.enterInlineAssemblyStatementFlag) {
        listener.enterInlineAssemblyStatementFlag(this);
      }
    }
    exitRule(listener) {
      if (listener.exitInlineAssemblyStatementFlag) {
        listener.exitInlineAssemblyStatementFlag(this);
      }
    }
    accept(visitor) {
      if (visitor.visitInlineAssemblyStatementFlag) {
        return visitor.visitInlineAssemblyStatementFlag(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var DoWhileStatementContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    statement() {
      return this.getTypedRuleContext(StatementContext, 0);
    }
    expression() {
      return this.getTypedRuleContext(ExpressionContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_doWhileStatement;
    }
    enterRule(listener) {
      if (listener.enterDoWhileStatement) {
        listener.enterDoWhileStatement(this);
      }
    }
    exitRule(listener) {
      if (listener.exitDoWhileStatement) {
        listener.exitDoWhileStatement(this);
      }
    }
    accept(visitor) {
      if (visitor.visitDoWhileStatement) {
        return visitor.visitDoWhileStatement(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var ContinueStatementContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    ContinueKeyword() {
      return this.getToken(SolidityParser.ContinueKeyword, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_continueStatement;
    }
    enterRule(listener) {
      if (listener.enterContinueStatement) {
        listener.enterContinueStatement(this);
      }
    }
    exitRule(listener) {
      if (listener.exitContinueStatement) {
        listener.exitContinueStatement(this);
      }
    }
    accept(visitor) {
      if (visitor.visitContinueStatement) {
        return visitor.visitContinueStatement(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var BreakStatementContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    BreakKeyword() {
      return this.getToken(SolidityParser.BreakKeyword, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_breakStatement;
    }
    enterRule(listener) {
      if (listener.enterBreakStatement) {
        listener.enterBreakStatement(this);
      }
    }
    exitRule(listener) {
      if (listener.exitBreakStatement) {
        listener.exitBreakStatement(this);
      }
    }
    accept(visitor) {
      if (visitor.visitBreakStatement) {
        return visitor.visitBreakStatement(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var ReturnStatementContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    expression() {
      return this.getTypedRuleContext(ExpressionContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_returnStatement;
    }
    enterRule(listener) {
      if (listener.enterReturnStatement) {
        listener.enterReturnStatement(this);
      }
    }
    exitRule(listener) {
      if (listener.exitReturnStatement) {
        listener.exitReturnStatement(this);
      }
    }
    accept(visitor) {
      if (visitor.visitReturnStatement) {
        return visitor.visitReturnStatement(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var ThrowStatementContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    get ruleIndex() {
      return SolidityParser.RULE_throwStatement;
    }
    enterRule(listener) {
      if (listener.enterThrowStatement) {
        listener.enterThrowStatement(this);
      }
    }
    exitRule(listener) {
      if (listener.exitThrowStatement) {
        listener.exitThrowStatement(this);
      }
    }
    accept(visitor) {
      if (visitor.visitThrowStatement) {
        return visitor.visitThrowStatement(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var EmitStatementContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    functionCall() {
      return this.getTypedRuleContext(FunctionCallContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_emitStatement;
    }
    enterRule(listener) {
      if (listener.enterEmitStatement) {
        listener.enterEmitStatement(this);
      }
    }
    exitRule(listener) {
      if (listener.exitEmitStatement) {
        listener.exitEmitStatement(this);
      }
    }
    accept(visitor) {
      if (visitor.visitEmitStatement) {
        return visitor.visitEmitStatement(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var RevertStatementContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    functionCall() {
      return this.getTypedRuleContext(FunctionCallContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_revertStatement;
    }
    enterRule(listener) {
      if (listener.enterRevertStatement) {
        listener.enterRevertStatement(this);
      }
    }
    exitRule(listener) {
      if (listener.exitRevertStatement) {
        listener.exitRevertStatement(this);
      }
    }
    accept(visitor) {
      if (visitor.visitRevertStatement) {
        return visitor.visitRevertStatement(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var VariableDeclarationStatementContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifierList() {
      return this.getTypedRuleContext(IdentifierListContext, 0);
    }
    variableDeclaration() {
      return this.getTypedRuleContext(VariableDeclarationContext, 0);
    }
    variableDeclarationList() {
      return this.getTypedRuleContext(VariableDeclarationListContext, 0);
    }
    expression() {
      return this.getTypedRuleContext(ExpressionContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_variableDeclarationStatement;
    }
    enterRule(listener) {
      if (listener.enterVariableDeclarationStatement) {
        listener.enterVariableDeclarationStatement(this);
      }
    }
    exitRule(listener) {
      if (listener.exitVariableDeclarationStatement) {
        listener.exitVariableDeclarationStatement(this);
      }
    }
    accept(visitor) {
      if (visitor.visitVariableDeclarationStatement) {
        return visitor.visitVariableDeclarationStatement(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var VariableDeclarationListContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    variableDeclaration_list() {
      return this.getTypedRuleContexts(VariableDeclarationContext);
    }
    variableDeclaration(i2) {
      return this.getTypedRuleContext(VariableDeclarationContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_variableDeclarationList;
    }
    enterRule(listener) {
      if (listener.enterVariableDeclarationList) {
        listener.enterVariableDeclarationList(this);
      }
    }
    exitRule(listener) {
      if (listener.exitVariableDeclarationList) {
        listener.exitVariableDeclarationList(this);
      }
    }
    accept(visitor) {
      if (visitor.visitVariableDeclarationList) {
        return visitor.visitVariableDeclarationList(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var IdentifierListContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifier_list() {
      return this.getTypedRuleContexts(IdentifierContext);
    }
    identifier(i2) {
      return this.getTypedRuleContext(IdentifierContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_identifierList;
    }
    enterRule(listener) {
      if (listener.enterIdentifierList) {
        listener.enterIdentifierList(this);
      }
    }
    exitRule(listener) {
      if (listener.exitIdentifierList) {
        listener.exitIdentifierList(this);
      }
    }
    accept(visitor) {
      if (visitor.visitIdentifierList) {
        return visitor.visitIdentifierList(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var ElementaryTypeNameContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    Int() {
      return this.getToken(SolidityParser.Int, 0);
    }
    Uint() {
      return this.getToken(SolidityParser.Uint, 0);
    }
    Byte() {
      return this.getToken(SolidityParser.Byte, 0);
    }
    Fixed() {
      return this.getToken(SolidityParser.Fixed, 0);
    }
    Ufixed() {
      return this.getToken(SolidityParser.Ufixed, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_elementaryTypeName;
    }
    enterRule(listener) {
      if (listener.enterElementaryTypeName) {
        listener.enterElementaryTypeName(this);
      }
    }
    exitRule(listener) {
      if (listener.exitElementaryTypeName) {
        listener.exitElementaryTypeName(this);
      }
    }
    accept(visitor) {
      if (visitor.visitElementaryTypeName) {
        return visitor.visitElementaryTypeName(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var ExpressionContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    typeName() {
      return this.getTypedRuleContext(TypeNameContext, 0);
    }
    expression_list() {
      return this.getTypedRuleContexts(ExpressionContext);
    }
    expression(i2) {
      return this.getTypedRuleContext(ExpressionContext, i2);
    }
    primaryExpression() {
      return this.getTypedRuleContext(PrimaryExpressionContext, 0);
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    nameValueList() {
      return this.getTypedRuleContext(NameValueListContext, 0);
    }
    functionCallArguments() {
      return this.getTypedRuleContext(FunctionCallArgumentsContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_expression;
    }
    enterRule(listener) {
      if (listener.enterExpression) {
        listener.enterExpression(this);
      }
    }
    exitRule(listener) {
      if (listener.exitExpression) {
        listener.exitExpression(this);
      }
    }
    accept(visitor) {
      if (visitor.visitExpression) {
        return visitor.visitExpression(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var PrimaryExpressionContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    BooleanLiteral() {
      return this.getToken(SolidityParser.BooleanLiteral, 0);
    }
    numberLiteral() {
      return this.getTypedRuleContext(NumberLiteralContext, 0);
    }
    hexLiteral() {
      return this.getTypedRuleContext(HexLiteralContext, 0);
    }
    stringLiteral() {
      return this.getTypedRuleContext(StringLiteralContext, 0);
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    TypeKeyword() {
      return this.getToken(SolidityParser.TypeKeyword, 0);
    }
    PayableKeyword() {
      return this.getToken(SolidityParser.PayableKeyword, 0);
    }
    tupleExpression() {
      return this.getTypedRuleContext(TupleExpressionContext, 0);
    }
    typeName() {
      return this.getTypedRuleContext(TypeNameContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_primaryExpression;
    }
    enterRule(listener) {
      if (listener.enterPrimaryExpression) {
        listener.enterPrimaryExpression(this);
      }
    }
    exitRule(listener) {
      if (listener.exitPrimaryExpression) {
        listener.exitPrimaryExpression(this);
      }
    }
    accept(visitor) {
      if (visitor.visitPrimaryExpression) {
        return visitor.visitPrimaryExpression(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var ExpressionListContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    expression_list() {
      return this.getTypedRuleContexts(ExpressionContext);
    }
    expression(i2) {
      return this.getTypedRuleContext(ExpressionContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_expressionList;
    }
    enterRule(listener) {
      if (listener.enterExpressionList) {
        listener.enterExpressionList(this);
      }
    }
    exitRule(listener) {
      if (listener.exitExpressionList) {
        listener.exitExpressionList(this);
      }
    }
    accept(visitor) {
      if (visitor.visitExpressionList) {
        return visitor.visitExpressionList(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var NameValueListContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    nameValue_list() {
      return this.getTypedRuleContexts(NameValueContext);
    }
    nameValue(i2) {
      return this.getTypedRuleContext(NameValueContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_nameValueList;
    }
    enterRule(listener) {
      if (listener.enterNameValueList) {
        listener.enterNameValueList(this);
      }
    }
    exitRule(listener) {
      if (listener.exitNameValueList) {
        listener.exitNameValueList(this);
      }
    }
    accept(visitor) {
      if (visitor.visitNameValueList) {
        return visitor.visitNameValueList(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var NameValueContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    expression() {
      return this.getTypedRuleContext(ExpressionContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_nameValue;
    }
    enterRule(listener) {
      if (listener.enterNameValue) {
        listener.enterNameValue(this);
      }
    }
    exitRule(listener) {
      if (listener.exitNameValue) {
        listener.exitNameValue(this);
      }
    }
    accept(visitor) {
      if (visitor.visitNameValue) {
        return visitor.visitNameValue(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var FunctionCallArgumentsContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    nameValueList() {
      return this.getTypedRuleContext(NameValueListContext, 0);
    }
    expressionList() {
      return this.getTypedRuleContext(ExpressionListContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_functionCallArguments;
    }
    enterRule(listener) {
      if (listener.enterFunctionCallArguments) {
        listener.enterFunctionCallArguments(this);
      }
    }
    exitRule(listener) {
      if (listener.exitFunctionCallArguments) {
        listener.exitFunctionCallArguments(this);
      }
    }
    accept(visitor) {
      if (visitor.visitFunctionCallArguments) {
        return visitor.visitFunctionCallArguments(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var FunctionCallContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    expression() {
      return this.getTypedRuleContext(ExpressionContext, 0);
    }
    functionCallArguments() {
      return this.getTypedRuleContext(FunctionCallArgumentsContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_functionCall;
    }
    enterRule(listener) {
      if (listener.enterFunctionCall) {
        listener.enterFunctionCall(this);
      }
    }
    exitRule(listener) {
      if (listener.exitFunctionCall) {
        listener.exitFunctionCall(this);
      }
    }
    accept(visitor) {
      if (visitor.visitFunctionCall) {
        return visitor.visitFunctionCall(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var AssemblyBlockContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    assemblyItem_list() {
      return this.getTypedRuleContexts(AssemblyItemContext);
    }
    assemblyItem(i2) {
      return this.getTypedRuleContext(AssemblyItemContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_assemblyBlock;
    }
    enterRule(listener) {
      if (listener.enterAssemblyBlock) {
        listener.enterAssemblyBlock(this);
      }
    }
    exitRule(listener) {
      if (listener.exitAssemblyBlock) {
        listener.exitAssemblyBlock(this);
      }
    }
    accept(visitor) {
      if (visitor.visitAssemblyBlock) {
        return visitor.visitAssemblyBlock(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var AssemblyItemContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    assemblyBlock() {
      return this.getTypedRuleContext(AssemblyBlockContext, 0);
    }
    assemblyExpression() {
      return this.getTypedRuleContext(AssemblyExpressionContext, 0);
    }
    assemblyLocalDefinition() {
      return this.getTypedRuleContext(AssemblyLocalDefinitionContext, 0);
    }
    assemblyAssignment() {
      return this.getTypedRuleContext(AssemblyAssignmentContext, 0);
    }
    assemblyStackAssignment() {
      return this.getTypedRuleContext(AssemblyStackAssignmentContext, 0);
    }
    labelDefinition() {
      return this.getTypedRuleContext(LabelDefinitionContext, 0);
    }
    assemblySwitch() {
      return this.getTypedRuleContext(AssemblySwitchContext, 0);
    }
    assemblyFunctionDefinition() {
      return this.getTypedRuleContext(AssemblyFunctionDefinitionContext, 0);
    }
    assemblyFor() {
      return this.getTypedRuleContext(AssemblyForContext, 0);
    }
    assemblyIf() {
      return this.getTypedRuleContext(AssemblyIfContext, 0);
    }
    BreakKeyword() {
      return this.getToken(SolidityParser.BreakKeyword, 0);
    }
    ContinueKeyword() {
      return this.getToken(SolidityParser.ContinueKeyword, 0);
    }
    LeaveKeyword() {
      return this.getToken(SolidityParser.LeaveKeyword, 0);
    }
    numberLiteral() {
      return this.getTypedRuleContext(NumberLiteralContext, 0);
    }
    stringLiteral() {
      return this.getTypedRuleContext(StringLiteralContext, 0);
    }
    hexLiteral() {
      return this.getTypedRuleContext(HexLiteralContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_assemblyItem;
    }
    enterRule(listener) {
      if (listener.enterAssemblyItem) {
        listener.enterAssemblyItem(this);
      }
    }
    exitRule(listener) {
      if (listener.exitAssemblyItem) {
        listener.exitAssemblyItem(this);
      }
    }
    accept(visitor) {
      if (visitor.visitAssemblyItem) {
        return visitor.visitAssemblyItem(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var AssemblyExpressionContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    assemblyCall() {
      return this.getTypedRuleContext(AssemblyCallContext, 0);
    }
    assemblyLiteral() {
      return this.getTypedRuleContext(AssemblyLiteralContext, 0);
    }
    assemblyMember() {
      return this.getTypedRuleContext(AssemblyMemberContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_assemblyExpression;
    }
    enterRule(listener) {
      if (listener.enterAssemblyExpression) {
        listener.enterAssemblyExpression(this);
      }
    }
    exitRule(listener) {
      if (listener.exitAssemblyExpression) {
        listener.exitAssemblyExpression(this);
      }
    }
    accept(visitor) {
      if (visitor.visitAssemblyExpression) {
        return visitor.visitAssemblyExpression(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var AssemblyMemberContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifier_list() {
      return this.getTypedRuleContexts(IdentifierContext);
    }
    identifier(i2) {
      return this.getTypedRuleContext(IdentifierContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_assemblyMember;
    }
    enterRule(listener) {
      if (listener.enterAssemblyMember) {
        listener.enterAssemblyMember(this);
      }
    }
    exitRule(listener) {
      if (listener.exitAssemblyMember) {
        listener.exitAssemblyMember(this);
      }
    }
    accept(visitor) {
      if (visitor.visitAssemblyMember) {
        return visitor.visitAssemblyMember(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var AssemblyCallContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    assemblyExpression_list() {
      return this.getTypedRuleContexts(AssemblyExpressionContext);
    }
    assemblyExpression(i2) {
      return this.getTypedRuleContext(AssemblyExpressionContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_assemblyCall;
    }
    enterRule(listener) {
      if (listener.enterAssemblyCall) {
        listener.enterAssemblyCall(this);
      }
    }
    exitRule(listener) {
      if (listener.exitAssemblyCall) {
        listener.exitAssemblyCall(this);
      }
    }
    accept(visitor) {
      if (visitor.visitAssemblyCall) {
        return visitor.visitAssemblyCall(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var AssemblyLocalDefinitionContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    assemblyIdentifierOrList() {
      return this.getTypedRuleContext(AssemblyIdentifierOrListContext, 0);
    }
    assemblyExpression() {
      return this.getTypedRuleContext(AssemblyExpressionContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_assemblyLocalDefinition;
    }
    enterRule(listener) {
      if (listener.enterAssemblyLocalDefinition) {
        listener.enterAssemblyLocalDefinition(this);
      }
    }
    exitRule(listener) {
      if (listener.exitAssemblyLocalDefinition) {
        listener.exitAssemblyLocalDefinition(this);
      }
    }
    accept(visitor) {
      if (visitor.visitAssemblyLocalDefinition) {
        return visitor.visitAssemblyLocalDefinition(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var AssemblyAssignmentContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    assemblyIdentifierOrList() {
      return this.getTypedRuleContext(AssemblyIdentifierOrListContext, 0);
    }
    assemblyExpression() {
      return this.getTypedRuleContext(AssemblyExpressionContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_assemblyAssignment;
    }
    enterRule(listener) {
      if (listener.enterAssemblyAssignment) {
        listener.enterAssemblyAssignment(this);
      }
    }
    exitRule(listener) {
      if (listener.exitAssemblyAssignment) {
        listener.exitAssemblyAssignment(this);
      }
    }
    accept(visitor) {
      if (visitor.visitAssemblyAssignment) {
        return visitor.visitAssemblyAssignment(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var AssemblyIdentifierOrListContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    assemblyMember() {
      return this.getTypedRuleContext(AssemblyMemberContext, 0);
    }
    assemblyIdentifierList() {
      return this.getTypedRuleContext(AssemblyIdentifierListContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_assemblyIdentifierOrList;
    }
    enterRule(listener) {
      if (listener.enterAssemblyIdentifierOrList) {
        listener.enterAssemblyIdentifierOrList(this);
      }
    }
    exitRule(listener) {
      if (listener.exitAssemblyIdentifierOrList) {
        listener.exitAssemblyIdentifierOrList(this);
      }
    }
    accept(visitor) {
      if (visitor.visitAssemblyIdentifierOrList) {
        return visitor.visitAssemblyIdentifierOrList(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var AssemblyIdentifierListContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifier_list() {
      return this.getTypedRuleContexts(IdentifierContext);
    }
    identifier(i2) {
      return this.getTypedRuleContext(IdentifierContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_assemblyIdentifierList;
    }
    enterRule(listener) {
      if (listener.enterAssemblyIdentifierList) {
        listener.enterAssemblyIdentifierList(this);
      }
    }
    exitRule(listener) {
      if (listener.exitAssemblyIdentifierList) {
        listener.exitAssemblyIdentifierList(this);
      }
    }
    accept(visitor) {
      if (visitor.visitAssemblyIdentifierList) {
        return visitor.visitAssemblyIdentifierList(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var AssemblyStackAssignmentContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    assemblyExpression() {
      return this.getTypedRuleContext(AssemblyExpressionContext, 0);
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_assemblyStackAssignment;
    }
    enterRule(listener) {
      if (listener.enterAssemblyStackAssignment) {
        listener.enterAssemblyStackAssignment(this);
      }
    }
    exitRule(listener) {
      if (listener.exitAssemblyStackAssignment) {
        listener.exitAssemblyStackAssignment(this);
      }
    }
    accept(visitor) {
      if (visitor.visitAssemblyStackAssignment) {
        return visitor.visitAssemblyStackAssignment(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var LabelDefinitionContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_labelDefinition;
    }
    enterRule(listener) {
      if (listener.enterLabelDefinition) {
        listener.enterLabelDefinition(this);
      }
    }
    exitRule(listener) {
      if (listener.exitLabelDefinition) {
        listener.exitLabelDefinition(this);
      }
    }
    accept(visitor) {
      if (visitor.visitLabelDefinition) {
        return visitor.visitLabelDefinition(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var AssemblySwitchContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    assemblyExpression() {
      return this.getTypedRuleContext(AssemblyExpressionContext, 0);
    }
    assemblyCase_list() {
      return this.getTypedRuleContexts(AssemblyCaseContext);
    }
    assemblyCase(i2) {
      return this.getTypedRuleContext(AssemblyCaseContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_assemblySwitch;
    }
    enterRule(listener) {
      if (listener.enterAssemblySwitch) {
        listener.enterAssemblySwitch(this);
      }
    }
    exitRule(listener) {
      if (listener.exitAssemblySwitch) {
        listener.exitAssemblySwitch(this);
      }
    }
    accept(visitor) {
      if (visitor.visitAssemblySwitch) {
        return visitor.visitAssemblySwitch(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var AssemblyCaseContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    assemblyLiteral() {
      return this.getTypedRuleContext(AssemblyLiteralContext, 0);
    }
    assemblyBlock() {
      return this.getTypedRuleContext(AssemblyBlockContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_assemblyCase;
    }
    enterRule(listener) {
      if (listener.enterAssemblyCase) {
        listener.enterAssemblyCase(this);
      }
    }
    exitRule(listener) {
      if (listener.exitAssemblyCase) {
        listener.exitAssemblyCase(this);
      }
    }
    accept(visitor) {
      if (visitor.visitAssemblyCase) {
        return visitor.visitAssemblyCase(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var AssemblyFunctionDefinitionContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    identifier() {
      return this.getTypedRuleContext(IdentifierContext, 0);
    }
    assemblyBlock() {
      return this.getTypedRuleContext(AssemblyBlockContext, 0);
    }
    assemblyIdentifierList() {
      return this.getTypedRuleContext(AssemblyIdentifierListContext, 0);
    }
    assemblyFunctionReturns() {
      return this.getTypedRuleContext(AssemblyFunctionReturnsContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_assemblyFunctionDefinition;
    }
    enterRule(listener) {
      if (listener.enterAssemblyFunctionDefinition) {
        listener.enterAssemblyFunctionDefinition(this);
      }
    }
    exitRule(listener) {
      if (listener.exitAssemblyFunctionDefinition) {
        listener.exitAssemblyFunctionDefinition(this);
      }
    }
    accept(visitor) {
      if (visitor.visitAssemblyFunctionDefinition) {
        return visitor.visitAssemblyFunctionDefinition(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var AssemblyFunctionReturnsContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    assemblyIdentifierList() {
      return this.getTypedRuleContext(AssemblyIdentifierListContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_assemblyFunctionReturns;
    }
    enterRule(listener) {
      if (listener.enterAssemblyFunctionReturns) {
        listener.enterAssemblyFunctionReturns(this);
      }
    }
    exitRule(listener) {
      if (listener.exitAssemblyFunctionReturns) {
        listener.exitAssemblyFunctionReturns(this);
      }
    }
    accept(visitor) {
      if (visitor.visitAssemblyFunctionReturns) {
        return visitor.visitAssemblyFunctionReturns(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var AssemblyForContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    assemblyExpression_list() {
      return this.getTypedRuleContexts(AssemblyExpressionContext);
    }
    assemblyExpression(i2) {
      return this.getTypedRuleContext(AssemblyExpressionContext, i2);
    }
    assemblyBlock_list() {
      return this.getTypedRuleContexts(AssemblyBlockContext);
    }
    assemblyBlock(i2) {
      return this.getTypedRuleContext(AssemblyBlockContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_assemblyFor;
    }
    enterRule(listener) {
      if (listener.enterAssemblyFor) {
        listener.enterAssemblyFor(this);
      }
    }
    exitRule(listener) {
      if (listener.exitAssemblyFor) {
        listener.exitAssemblyFor(this);
      }
    }
    accept(visitor) {
      if (visitor.visitAssemblyFor) {
        return visitor.visitAssemblyFor(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var AssemblyIfContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    assemblyExpression() {
      return this.getTypedRuleContext(AssemblyExpressionContext, 0);
    }
    assemblyBlock() {
      return this.getTypedRuleContext(AssemblyBlockContext, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_assemblyIf;
    }
    enterRule(listener) {
      if (listener.enterAssemblyIf) {
        listener.enterAssemblyIf(this);
      }
    }
    exitRule(listener) {
      if (listener.exitAssemblyIf) {
        listener.exitAssemblyIf(this);
      }
    }
    accept(visitor) {
      if (visitor.visitAssemblyIf) {
        return visitor.visitAssemblyIf(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var AssemblyLiteralContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    stringLiteral() {
      return this.getTypedRuleContext(StringLiteralContext, 0);
    }
    DecimalNumber() {
      return this.getToken(SolidityParser.DecimalNumber, 0);
    }
    HexNumber() {
      return this.getToken(SolidityParser.HexNumber, 0);
    }
    hexLiteral() {
      return this.getTypedRuleContext(HexLiteralContext, 0);
    }
    BooleanLiteral() {
      return this.getToken(SolidityParser.BooleanLiteral, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_assemblyLiteral;
    }
    enterRule(listener) {
      if (listener.enterAssemblyLiteral) {
        listener.enterAssemblyLiteral(this);
      }
    }
    exitRule(listener) {
      if (listener.exitAssemblyLiteral) {
        listener.exitAssemblyLiteral(this);
      }
    }
    accept(visitor) {
      if (visitor.visitAssemblyLiteral) {
        return visitor.visitAssemblyLiteral(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var TupleExpressionContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    expression_list() {
      return this.getTypedRuleContexts(ExpressionContext);
    }
    expression(i2) {
      return this.getTypedRuleContext(ExpressionContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_tupleExpression;
    }
    enterRule(listener) {
      if (listener.enterTupleExpression) {
        listener.enterTupleExpression(this);
      }
    }
    exitRule(listener) {
      if (listener.exitTupleExpression) {
        listener.exitTupleExpression(this);
      }
    }
    accept(visitor) {
      if (visitor.visitTupleExpression) {
        return visitor.visitTupleExpression(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var NumberLiteralContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    DecimalNumber() {
      return this.getToken(SolidityParser.DecimalNumber, 0);
    }
    HexNumber() {
      return this.getToken(SolidityParser.HexNumber, 0);
    }
    NumberUnit() {
      return this.getToken(SolidityParser.NumberUnit, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_numberLiteral;
    }
    enterRule(listener) {
      if (listener.enterNumberLiteral) {
        listener.enterNumberLiteral(this);
      }
    }
    exitRule(listener) {
      if (listener.exitNumberLiteral) {
        listener.exitNumberLiteral(this);
      }
    }
    accept(visitor) {
      if (visitor.visitNumberLiteral) {
        return visitor.visitNumberLiteral(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var IdentifierContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    ReceiveKeyword() {
      return this.getToken(SolidityParser.ReceiveKeyword, 0);
    }
    GlobalKeyword() {
      return this.getToken(SolidityParser.GlobalKeyword, 0);
    }
    ConstructorKeyword() {
      return this.getToken(SolidityParser.ConstructorKeyword, 0);
    }
    PayableKeyword() {
      return this.getToken(SolidityParser.PayableKeyword, 0);
    }
    LeaveKeyword() {
      return this.getToken(SolidityParser.LeaveKeyword, 0);
    }
    Identifier() {
      return this.getToken(SolidityParser.Identifier, 0);
    }
    get ruleIndex() {
      return SolidityParser.RULE_identifier;
    }
    enterRule(listener) {
      if (listener.enterIdentifier) {
        listener.enterIdentifier(this);
      }
    }
    exitRule(listener) {
      if (listener.exitIdentifier) {
        listener.exitIdentifier(this);
      }
    }
    accept(visitor) {
      if (visitor.visitIdentifier) {
        return visitor.visitIdentifier(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var HexLiteralContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    HexLiteralFragment_list() {
      return this.getTokens(SolidityParser.HexLiteralFragment);
    }
    HexLiteralFragment(i2) {
      return this.getToken(SolidityParser.HexLiteralFragment, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_hexLiteral;
    }
    enterRule(listener) {
      if (listener.enterHexLiteral) {
        listener.enterHexLiteral(this);
      }
    }
    exitRule(listener) {
      if (listener.exitHexLiteral) {
        listener.exitHexLiteral(this);
      }
    }
    accept(visitor) {
      if (visitor.visitHexLiteral) {
        return visitor.visitHexLiteral(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var OverrideSpecifierContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    userDefinedTypeName_list() {
      return this.getTypedRuleContexts(UserDefinedTypeNameContext);
    }
    userDefinedTypeName(i2) {
      return this.getTypedRuleContext(UserDefinedTypeNameContext, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_overrideSpecifier;
    }
    enterRule(listener) {
      if (listener.enterOverrideSpecifier) {
        listener.enterOverrideSpecifier(this);
      }
    }
    exitRule(listener) {
      if (listener.exitOverrideSpecifier) {
        listener.exitOverrideSpecifier(this);
      }
    }
    accept(visitor) {
      if (visitor.visitOverrideSpecifier) {
        return visitor.visitOverrideSpecifier(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };
  var StringLiteralContext = class extends L {
    constructor(parser, parent, invokingState) {
      super(parent, invokingState);
      this.parser = parser;
    }
    StringLiteralFragment_list() {
      return this.getTokens(SolidityParser.StringLiteralFragment);
    }
    StringLiteralFragment(i2) {
      return this.getToken(SolidityParser.StringLiteralFragment, i2);
    }
    get ruleIndex() {
      return SolidityParser.RULE_stringLiteral;
    }
    enterRule(listener) {
      if (listener.enterStringLiteral) {
        listener.enterStringLiteral(this);
      }
    }
    exitRule(listener) {
      if (listener.exitStringLiteral) {
        listener.exitStringLiteral(this);
      }
    }
    accept(visitor) {
      if (visitor.visitStringLiteral) {
        return visitor.visitStringLiteral(this);
      } else {
        return visitor.visitChildren(this);
      }
    }
  };

  // src/ast-types.ts
  var astNodeTypes = [
    "SourceUnit",
    "PragmaDirective",
    "ImportDirective",
    "ContractDefinition",
    "InheritanceSpecifier",
    "StateVariableDeclaration",
    "UsingForDeclaration",
    "StructDefinition",
    "ModifierDefinition",
    "ModifierInvocation",
    "FunctionDefinition",
    "EventDefinition",
    "CustomErrorDefinition",
    "RevertStatement",
    "EnumValue",
    "EnumDefinition",
    "VariableDeclaration",
    "UserDefinedTypeName",
    "Mapping",
    "ArrayTypeName",
    "FunctionTypeName",
    "Block",
    "ExpressionStatement",
    "IfStatement",
    "WhileStatement",
    "ForStatement",
    "InlineAssemblyStatement",
    "DoWhileStatement",
    "ContinueStatement",
    "Break",
    "Continue",
    "BreakStatement",
    "ReturnStatement",
    "EmitStatement",
    "ThrowStatement",
    "VariableDeclarationStatement",
    "ElementaryTypeName",
    "FunctionCall",
    "AssemblyBlock",
    "AssemblyCall",
    "AssemblyLocalDefinition",
    "AssemblyAssignment",
    "AssemblyStackAssignment",
    "LabelDefinition",
    "AssemblySwitch",
    "AssemblyCase",
    "AssemblyFunctionDefinition",
    "AssemblyFor",
    "AssemblyIf",
    "TupleExpression",
    "NameValueExpression",
    "BooleanLiteral",
    "NumberLiteral",
    "Identifier",
    "BinaryOperation",
    "UnaryOperation",
    "NewExpression",
    "Conditional",
    "StringLiteral",
    "HexLiteral",
    "HexNumber",
    "DecimalNumber",
    "MemberAccess",
    "IndexAccess",
    "IndexRangeAccess",
    "NameValueList",
    "UncheckedStatement",
    "TryStatement",
    "CatchClause",
    "FileLevelConstant",
    "AssemblyMemberAccess",
    "TypeDefinition"
  ];
  var binaryOpValues = [
    "+",
    "-",
    "*",
    "/",
    "**",
    "%",
    "<<",
    ">>",
    "&&",
    "||",
    "&",
    "^",
    "<",
    ">",
    "<=",
    ">=",
    "==",
    "!=",
    "=",
    "^=",
    "&=",
    "<<=",
    ">>=",
    "+=",
    "-=",
    "*=",
    "/=",
    "%=",
    "|",
    "|="
  ];
  var unaryOpValues = [
    "-",
    "+",
    "++",
    "--",
    "~",
    "after",
    "delete",
    "!"
  ];

  // src/ASTBuilder.ts
  var ASTBuilder = class extends N {
    constructor(options) {
      super();
      this.options = options;
      this.result = null;
    }
    defaultResult() {
      throw new Error("Unknown node");
    }
    aggregateResult() {
      return {type: ""};
    }
    visitSourceUnit(ctx) {
      const children = ctx.children ?? [];
      const node = {
        type: "SourceUnit",
        children: children.slice(0, -1).map((child) => this.visit(child))
      };
      const result = this._addMeta(node, ctx);
      this.result = result;
      return result;
    }
    visitContractPart(ctx) {
      return this.visit(ctx.getChild(0));
    }
    visitContractDefinition(ctx) {
      const name = this._toText(ctx.identifier());
      const kind = this._toText(ctx.getChild(0));
      this._currentContract = name;
      const node = {
        type: "ContractDefinition",
        name,
        baseContracts: ctx.inheritanceSpecifier_list().map((x2) => this.visitInheritanceSpecifier(x2)),
        subNodes: ctx.contractPart_list().map((x2) => this.visit(x2)),
        kind
      };
      return this._addMeta(node, ctx);
    }
    visitStateVariableDeclaration(ctx) {
      const type = this.visitTypeName(ctx.typeName());
      const iden = ctx.identifier();
      const name = this._toText(iden);
      let expression = null;
      const ctxExpression = ctx.expression();
      if (ctxExpression) {
        expression = this.visitExpression(ctxExpression);
      }
      let visibility = "default";
      if (ctx.InternalKeyword_list().length > 0) {
        visibility = "internal";
      } else if (ctx.PublicKeyword_list().length > 0) {
        visibility = "public";
      } else if (ctx.PrivateKeyword_list().length > 0) {
        visibility = "private";
      }
      let isDeclaredConst = false;
      if (ctx.ConstantKeyword_list().length > 0) {
        isDeclaredConst = true;
      }
      let override;
      const overrideSpecifier = ctx.overrideSpecifier_list();
      if (overrideSpecifier.length === 0) {
        override = null;
      } else {
        override = overrideSpecifier[0].userDefinedTypeName_list().map((x2) => this.visitUserDefinedTypeName(x2));
      }
      let isImmutable = false;
      if (ctx.ImmutableKeyword_list().length > 0) {
        isImmutable = true;
      }
      const decl = {
        type: "VariableDeclaration",
        typeName: type,
        name,
        identifier: this.visitIdentifier(iden),
        expression,
        visibility,
        isStateVar: true,
        isDeclaredConst,
        isIndexed: false,
        isImmutable,
        override,
        storageLocation: null
      };
      const node = {
        type: "StateVariableDeclaration",
        variables: [this._addMeta(decl, ctx)],
        initialValue: expression
      };
      return this._addMeta(node, ctx);
    }
    visitVariableDeclaration(ctx) {
      let storageLocation = null;
      const ctxStorageLocation = ctx.storageLocation();
      if (ctxStorageLocation) {
        storageLocation = this._toText(ctxStorageLocation);
      }
      const identifierCtx = ctx.identifier();
      const node = {
        type: "VariableDeclaration",
        typeName: this.visitTypeName(ctx.typeName()),
        name: this._toText(identifierCtx),
        identifier: this.visitIdentifier(identifierCtx),
        storageLocation,
        isStateVar: false,
        isIndexed: false,
        expression: null
      };
      return this._addMeta(node, ctx);
    }
    visitVariableDeclarationStatement(ctx) {
      let variables = [];
      const ctxVariableDeclaration = ctx.variableDeclaration();
      const ctxIdentifierList = ctx.identifierList();
      const ctxVariableDeclarationList = ctx.variableDeclarationList();
      if (ctxVariableDeclaration) {
        variables = [this.visitVariableDeclaration(ctxVariableDeclaration)];
      } else if (ctxIdentifierList) {
        variables = this.buildIdentifierList(ctxIdentifierList);
      } else if (ctxVariableDeclarationList) {
        variables = this.buildVariableDeclarationList(ctxVariableDeclarationList);
      }
      let initialValue = null;
      const ctxExpression = ctx.expression();
      if (ctxExpression) {
        initialValue = this.visitExpression(ctxExpression);
      }
      const node = {
        type: "VariableDeclarationStatement",
        variables,
        initialValue
      };
      return this._addMeta(node, ctx);
    }
    visitStatement(ctx) {
      return this.visit(ctx.getChild(0));
    }
    visitSimpleStatement(ctx) {
      return this.visit(ctx.getChild(0));
    }
    visitEventDefinition(ctx) {
      const parameters = ctx.eventParameterList().eventParameter_list().map((paramCtx) => {
        const type = this.visitTypeName(paramCtx.typeName());
        let name = null;
        const paramCtxIdentifier = paramCtx.identifier();
        if (paramCtxIdentifier) {
          name = this._toText(paramCtxIdentifier);
        }
        const node2 = {
          type: "VariableDeclaration",
          typeName: type,
          name,
          identifier: paramCtxIdentifier ? this.visitIdentifier(paramCtxIdentifier) : null,
          isStateVar: false,
          isIndexed: Boolean(paramCtx.IndexedKeyword()),
          storageLocation: null,
          expression: null
        };
        return this._addMeta(node2, paramCtx);
      });
      const node = {
        type: "EventDefinition",
        name: this._toText(ctx.identifier()),
        parameters,
        isAnonymous: Boolean(ctx.AnonymousKeyword())
      };
      return this._addMeta(node, ctx);
    }
    visitBlock(ctx) {
      const node = {
        type: "Block",
        statements: ctx.statement_list().map((x2) => this.visitStatement(x2))
      };
      return this._addMeta(node, ctx);
    }
    visitParameter(ctx) {
      let storageLocation = null;
      const ctxStorageLocation = ctx.storageLocation();
      if (ctxStorageLocation) {
        storageLocation = this._toText(ctxStorageLocation);
      }
      let name = null;
      const ctxIdentifier = ctx.identifier();
      if (ctxIdentifier) {
        name = this._toText(ctxIdentifier);
      }
      const node = {
        type: "VariableDeclaration",
        typeName: this.visitTypeName(ctx.typeName()),
        name,
        identifier: ctxIdentifier ? this.visitIdentifier(ctxIdentifier) : null,
        storageLocation,
        isStateVar: false,
        isIndexed: false,
        expression: null
      };
      return this._addMeta(node, ctx);
    }
    visitFunctionDefinition(ctx) {
      let isConstructor = false;
      let isFallback = false;
      let isReceiveEther = false;
      let isVirtual = false;
      let name = null;
      let parameters = [];
      let returnParameters = null;
      let visibility = "default";
      let block = null;
      const ctxBlock = ctx.block();
      if (ctxBlock) {
        block = this.visitBlock(ctxBlock);
      }
      const modifiers = ctx.modifierList().modifierInvocation_list().map((mod) => this.visitModifierInvocation(mod));
      let stateMutability = null;
      if (ctx.modifierList().stateMutability_list().length > 0) {
        stateMutability = this._stateMutabilityToText(ctx.modifierList().stateMutability(0));
      }
      const ctxReturnParameters = ctx.returnParameters();
      switch (this._toText(ctx.functionDescriptor().getChild(0))) {
        case "constructor":
          parameters = ctx.parameterList().parameter_list().map((x2) => this.visit(x2));
          if (ctx.modifierList().InternalKeyword_list().length > 0) {
            visibility = "internal";
          } else if (ctx.modifierList().PublicKeyword_list().length > 0) {
            visibility = "public";
          } else {
            visibility = "default";
          }
          isConstructor = true;
          break;
        case "fallback":
          parameters = ctx.parameterList().parameter_list().map((x2) => this.visit(x2));
          returnParameters = ctxReturnParameters ? this.visitReturnParameters(ctxReturnParameters) : null;
          visibility = "external";
          isFallback = true;
          break;
        case "receive":
          visibility = "external";
          isReceiveEther = true;
          break;
        case "function": {
          const identifier = ctx.functionDescriptor().identifier();
          name = identifier ? this._toText(identifier) : "";
          parameters = ctx.parameterList().parameter_list().map((x2) => this.visit(x2));
          returnParameters = ctxReturnParameters ? this.visitReturnParameters(ctxReturnParameters) : null;
          if (ctx.modifierList().ExternalKeyword_list().length > 0) {
            visibility = "external";
          } else if (ctx.modifierList().InternalKeyword_list().length > 0) {
            visibility = "internal";
          } else if (ctx.modifierList().PublicKeyword_list().length > 0) {
            visibility = "public";
          } else if (ctx.modifierList().PrivateKeyword_list().length > 0) {
            visibility = "private";
          }
          isConstructor = name === this._currentContract;
          isFallback = name === "";
          break;
        }
      }
      if (ctx.modifierList().VirtualKeyword_list().length > 0) {
        isVirtual = true;
      }
      let override;
      const overrideSpecifier = ctx.modifierList().overrideSpecifier_list();
      if (overrideSpecifier.length === 0) {
        override = null;
      } else {
        override = overrideSpecifier[0].userDefinedTypeName_list().map((x2) => this.visitUserDefinedTypeName(x2));
      }
      const node = {
        type: "FunctionDefinition",
        name,
        parameters,
        returnParameters,
        body: block,
        visibility,
        modifiers,
        override,
        isConstructor,
        isReceiveEther,
        isFallback,
        isVirtual,
        stateMutability
      };
      return this._addMeta(node, ctx);
    }
    visitEnumDefinition(ctx) {
      const node = {
        type: "EnumDefinition",
        name: this._toText(ctx.identifier()),
        members: ctx.enumValue_list().map((x2) => this.visitEnumValue(x2))
      };
      return this._addMeta(node, ctx);
    }
    visitEnumValue(ctx) {
      const node = {
        type: "EnumValue",
        name: this._toText(ctx.identifier())
      };
      return this._addMeta(node, ctx);
    }
    visitElementaryTypeName(ctx) {
      const node = {
        type: "ElementaryTypeName",
        name: this._toText(ctx),
        stateMutability: null
      };
      return this._addMeta(node, ctx);
    }
    visitIdentifier(ctx) {
      const node = {
        type: "Identifier",
        name: this._toText(ctx)
      };
      return this._addMeta(node, ctx);
    }
    visitTypeName(ctx) {
      if (ctx.children && ctx.children.length > 2) {
        let length = null;
        if (ctx.children.length === 4) {
          const expression = ctx.expression();
          if (expression === void 0 || expression === null) {
            throw new Error("Assertion error: a typeName with 4 children should have an expression");
          }
          length = this.visitExpression(expression);
        }
        const node = {
          type: "ArrayTypeName",
          baseTypeName: this.visitTypeName(ctx.typeName()),
          length
        };
        return this._addMeta(node, ctx);
      }
      if (ctx.children?.length === 2) {
        const node = {
          type: "ElementaryTypeName",
          name: this._toText(ctx.getChild(0)),
          stateMutability: this._toText(ctx.getChild(1))
        };
        return this._addMeta(node, ctx);
      }
      if (ctx.elementaryTypeName()) {
        return this.visitElementaryTypeName(ctx.elementaryTypeName());
      }
      if (ctx.userDefinedTypeName()) {
        return this.visitUserDefinedTypeName(ctx.userDefinedTypeName());
      }
      if (ctx.mapping()) {
        return this.visitMapping(ctx.mapping());
      }
      if (ctx.functionTypeName()) {
        return this.visitFunctionTypeName(ctx.functionTypeName());
      }
      throw new Error("Assertion error: unhandled type name case");
    }
    visitUserDefinedTypeName(ctx) {
      const node = {
        type: "UserDefinedTypeName",
        namePath: this._toText(ctx)
      };
      return this._addMeta(node, ctx);
    }
    visitUsingForDeclaration(ctx) {
      let typeName = null;
      const ctxTypeName = ctx.typeName();
      if (ctxTypeName) {
        typeName = this.visitTypeName(ctxTypeName);
      }
      const isGlobal = Boolean(ctx.GlobalKeyword());
      const usingForObjectCtx = ctx.usingForObject();
      const userDefinedTypeNameCtx = usingForObjectCtx.userDefinedTypeName();
      let node;
      if (userDefinedTypeNameCtx) {
        node = {
          type: "UsingForDeclaration",
          isGlobal,
          typeName,
          libraryName: this._toText(userDefinedTypeNameCtx),
          functions: [],
          operators: []
        };
      } else {
        const usingForObjectDirectives = usingForObjectCtx.usingForObjectDirective_list();
        const functions = [];
        const operators = [];
        for (const usingForObjectDirective of usingForObjectDirectives) {
          functions.push(this._toText(usingForObjectDirective.userDefinedTypeName()));
          const operator = usingForObjectDirective.userDefinableOperators();
          if (operator) {
            operators.push(this._toText(operator));
          } else {
            operators.push(null);
          }
        }
        node = {
          type: "UsingForDeclaration",
          isGlobal,
          typeName,
          libraryName: null,
          functions,
          operators
        };
      }
      return this._addMeta(node, ctx);
    }
    visitPragmaDirective(ctx) {
      const versionContext = ctx.pragmaValue().version();
      let value = this._toText(ctx.pragmaValue());
      if (versionContext?.children) {
        value = versionContext.children.map((x2) => this._toText(x2)).join(" ");
      }
      const node = {
        type: "PragmaDirective",
        name: this._toText(ctx.pragmaName()),
        value
      };
      return this._addMeta(node, ctx);
    }
    visitInheritanceSpecifier(ctx) {
      const exprList = ctx.expressionList();
      const args = exprList ? exprList.expression_list().map((x2) => this.visitExpression(x2)) : [];
      const node = {
        type: "InheritanceSpecifier",
        baseName: this.visitUserDefinedTypeName(ctx.userDefinedTypeName()),
        arguments: args
      };
      return this._addMeta(node, ctx);
    }
    visitModifierInvocation(ctx) {
      const exprList = ctx.expressionList();
      let args;
      if (exprList != null) {
        args = exprList.expression_list().map((x2) => this.visit(x2));
      } else if (ctx.children && ctx.children.length > 1) {
        args = [];
      } else {
        args = null;
      }
      const node = {
        type: "ModifierInvocation",
        name: this._toText(ctx.identifier()),
        arguments: args
      };
      return this._addMeta(node, ctx);
    }
    visitFunctionTypeName(ctx) {
      const parameterTypes = ctx.functionTypeParameterList(0).functionTypeParameter_list().map((typeCtx) => this.visitFunctionTypeParameter(typeCtx));
      let returnTypes = [];
      if (ctx.functionTypeParameterList_list().length > 1) {
        returnTypes = ctx.functionTypeParameterList(1).functionTypeParameter_list().map((typeCtx) => this.visitFunctionTypeParameter(typeCtx));
      }
      let visibility = "default";
      if (ctx.InternalKeyword_list().length > 0) {
        visibility = "internal";
      } else if (ctx.ExternalKeyword_list().length > 0) {
        visibility = "external";
      }
      let stateMutability = null;
      if (ctx.stateMutability_list().length > 0) {
        stateMutability = this._toText(ctx.stateMutability(0));
      }
      const node = {
        type: "FunctionTypeName",
        parameterTypes,
        returnTypes,
        visibility,
        stateMutability
      };
      return this._addMeta(node, ctx);
    }
    visitFunctionTypeParameter(ctx) {
      let storageLocation = null;
      if (ctx.storageLocation()) {
        storageLocation = this._toText(ctx.storageLocation());
      }
      const node = {
        type: "VariableDeclaration",
        typeName: this.visitTypeName(ctx.typeName()),
        name: null,
        identifier: null,
        storageLocation,
        isStateVar: false,
        isIndexed: false,
        expression: null
      };
      return this._addMeta(node, ctx);
    }
    visitThrowStatement(ctx) {
      const node = {
        type: "ThrowStatement"
      };
      return this._addMeta(node, ctx);
    }
    visitReturnStatement(ctx) {
      let expression = null;
      const ctxExpression = ctx.expression();
      if (ctxExpression) {
        expression = this.visitExpression(ctxExpression);
      }
      const node = {
        type: "ReturnStatement",
        expression
      };
      return this._addMeta(node, ctx);
    }
    visitEmitStatement(ctx) {
      const node = {
        type: "EmitStatement",
        eventCall: this.visitFunctionCall(ctx.functionCall())
      };
      return this._addMeta(node, ctx);
    }
    visitCustomErrorDefinition(ctx) {
      const node = {
        type: "CustomErrorDefinition",
        name: this._toText(ctx.identifier()),
        parameters: this.visitParameterList(ctx.parameterList())
      };
      return this._addMeta(node, ctx);
    }
    visitTypeDefinition(ctx) {
      const node = {
        type: "TypeDefinition",
        name: this._toText(ctx.identifier()),
        definition: this.visitElementaryTypeName(ctx.elementaryTypeName())
      };
      return this._addMeta(node, ctx);
    }
    visitRevertStatement(ctx) {
      const node = {
        type: "RevertStatement",
        revertCall: this.visitFunctionCall(ctx.functionCall())
      };
      return this._addMeta(node, ctx);
    }
    visitFunctionCall(ctx) {
      let args = [];
      const names = [];
      const identifiers = [];
      const ctxArgs = ctx.functionCallArguments();
      const ctxArgsExpressionList = ctxArgs.expressionList();
      const ctxArgsNameValueList = ctxArgs.nameValueList();
      if (ctxArgsExpressionList) {
        args = ctxArgsExpressionList.expression_list().map((exprCtx) => this.visitExpression(exprCtx));
      } else if (ctxArgsNameValueList) {
        for (const nameValue of ctxArgsNameValueList.nameValue_list()) {
          args.push(this.visitExpression(nameValue.expression()));
          names.push(this._toText(nameValue.identifier()));
          identifiers.push(this.visitIdentifier(nameValue.identifier()));
        }
      }
      const node = {
        type: "FunctionCall",
        expression: this.visitExpression(ctx.expression()),
        arguments: args,
        names,
        identifiers
      };
      return this._addMeta(node, ctx);
    }
    visitStructDefinition(ctx) {
      const node = {
        type: "StructDefinition",
        name: this._toText(ctx.identifier()),
        members: ctx.variableDeclaration_list().map((x2) => this.visitVariableDeclaration(x2))
      };
      return this._addMeta(node, ctx);
    }
    visitWhileStatement(ctx) {
      const node = {
        type: "WhileStatement",
        condition: this.visitExpression(ctx.expression()),
        body: this.visitStatement(ctx.statement())
      };
      return this._addMeta(node, ctx);
    }
    visitDoWhileStatement(ctx) {
      const node = {
        type: "DoWhileStatement",
        condition: this.visitExpression(ctx.expression()),
        body: this.visitStatement(ctx.statement())
      };
      return this._addMeta(node, ctx);
    }
    visitIfStatement(ctx) {
      const trueBody = this.visitStatement(ctx.statement(0));
      let falseBody = null;
      if (ctx.statement_list().length > 1) {
        falseBody = this.visitStatement(ctx.statement(1));
      }
      const node = {
        type: "IfStatement",
        condition: this.visitExpression(ctx.expression()),
        trueBody,
        falseBody
      };
      return this._addMeta(node, ctx);
    }
    visitTryStatement(ctx) {
      let returnParameters = null;
      const ctxReturnParameters = ctx.returnParameters();
      if (ctxReturnParameters) {
        returnParameters = this.visitReturnParameters(ctxReturnParameters);
      }
      const catchClauses = ctx.catchClause_list().map((exprCtx) => this.visitCatchClause(exprCtx));
      const node = {
        type: "TryStatement",
        expression: this.visitExpression(ctx.expression()),
        returnParameters,
        body: this.visitBlock(ctx.block()),
        catchClauses
      };
      return this._addMeta(node, ctx);
    }
    visitCatchClause(ctx) {
      let parameters = null;
      if (ctx.parameterList()) {
        parameters = this.visitParameterList(ctx.parameterList());
      }
      if (ctx.identifier() && this._toText(ctx.identifier()) !== "Error" && this._toText(ctx.identifier()) !== "Panic") {
        throw new Error('Expected "Error" or "Panic" identifier in catch clause');
      }
      let kind = null;
      const ctxIdentifier = ctx.identifier();
      if (ctxIdentifier) {
        kind = this._toText(ctxIdentifier);
      }
      const node = {
        type: "CatchClause",
        isReasonStringType: kind === "Error",
        kind,
        parameters,
        body: this.visitBlock(ctx.block())
      };
      return this._addMeta(node, ctx);
    }
    visitExpressionStatement(ctx) {
      if (!ctx) {
        return null;
      }
      const node = {
        type: "ExpressionStatement",
        expression: this.visitExpression(ctx.expression())
      };
      return this._addMeta(node, ctx);
    }
    visitNumberLiteral(ctx) {
      const number = this._toText(ctx.getChild(0));
      let subdenomination = null;
      if (ctx.children?.length === 2) {
        subdenomination = this._toText(ctx.getChild(1));
      }
      const node = {
        type: "NumberLiteral",
        number,
        subdenomination
      };
      return this._addMeta(node, ctx);
    }
    visitMappingKey(ctx) {
      if (ctx.elementaryTypeName()) {
        return this.visitElementaryTypeName(ctx.elementaryTypeName());
      } else if (ctx.userDefinedTypeName()) {
        return this.visitUserDefinedTypeName(ctx.userDefinedTypeName());
      } else {
        throw new Error("Expected MappingKey to have either elementaryTypeName or userDefinedTypeName");
      }
    }
    visitMapping(ctx) {
      const mappingKeyNameCtx = ctx.mappingKeyName();
      const mappingValueNameCtx = ctx.mappingValueName();
      const node = {
        type: "Mapping",
        keyType: this.visitMappingKey(ctx.mappingKey()),
        keyName: mappingKeyNameCtx ? this.visitIdentifier(mappingKeyNameCtx.identifier()) : null,
        valueType: this.visitTypeName(ctx.typeName()),
        valueName: mappingValueNameCtx ? this.visitIdentifier(mappingValueNameCtx.identifier()) : null
      };
      return this._addMeta(node, ctx);
    }
    visitModifierDefinition(ctx) {
      let parameters = null;
      if (ctx.parameterList()) {
        parameters = this.visitParameterList(ctx.parameterList());
      }
      let isVirtual = false;
      if (ctx.VirtualKeyword_list().length > 0) {
        isVirtual = true;
      }
      let override;
      const overrideSpecifier = ctx.overrideSpecifier_list();
      if (overrideSpecifier.length === 0) {
        override = null;
      } else {
        override = overrideSpecifier[0].userDefinedTypeName_list().map((x2) => this.visitUserDefinedTypeName(x2));
      }
      let body = null;
      const blockCtx = ctx.block();
      if (blockCtx) {
        body = this.visitBlock(blockCtx);
      }
      const node = {
        type: "ModifierDefinition",
        name: this._toText(ctx.identifier()),
        parameters,
        body,
        isVirtual,
        override
      };
      return this._addMeta(node, ctx);
    }
    visitUncheckedStatement(ctx) {
      const node = {
        type: "UncheckedStatement",
        block: this.visitBlock(ctx.block())
      };
      return this._addMeta(node, ctx);
    }
    visitExpression(ctx) {
      let op;
      switch (ctx.children.length) {
        case 1: {
          const primaryExpressionCtx = ctx.primaryExpression();
          if (primaryExpressionCtx === void 0 || primaryExpressionCtx === null) {
            throw new Error("Assertion error: primary expression should exist when children length is 1");
          }
          return this.visitPrimaryExpression(primaryExpressionCtx);
        }
        case 2:
          op = this._toText(ctx.getChild(0));
          if (op === "new") {
            const node = {
              type: "NewExpression",
              typeName: this.visitTypeName(ctx.typeName())
            };
            return this._addMeta(node, ctx);
          }
          if (unaryOpValues.includes(op)) {
            const node = {
              type: "UnaryOperation",
              operator: op,
              subExpression: this.visitExpression(ctx.expression(0)),
              isPrefix: true
            };
            return this._addMeta(node, ctx);
          }
          op = this._toText(ctx.getChild(1));
          if (["++", "--"].includes(op)) {
            const node = {
              type: "UnaryOperation",
              operator: op,
              subExpression: this.visitExpression(ctx.expression(0)),
              isPrefix: false
            };
            return this._addMeta(node, ctx);
          }
          break;
        case 3:
          if (this._toText(ctx.getChild(0)) === "(" && this._toText(ctx.getChild(2)) === ")") {
            const node = {
              type: "TupleExpression",
              components: [this.visitExpression(ctx.expression(0))],
              isArray: false
            };
            return this._addMeta(node, ctx);
          }
          op = this._toText(ctx.getChild(1));
          if (op === ".") {
            const node = {
              type: "MemberAccess",
              expression: this.visitExpression(ctx.expression(0)),
              memberName: this._toText(ctx.identifier())
            };
            return this._addMeta(node, ctx);
          }
          if (isBinOp(op)) {
            const node = {
              type: "BinaryOperation",
              operator: op,
              left: this.visitExpression(ctx.expression(0)),
              right: this.visitExpression(ctx.expression(1))
            };
            return this._addMeta(node, ctx);
          }
          break;
        case 4:
          if (this._toText(ctx.getChild(1)) === "(" && this._toText(ctx.getChild(3)) === ")") {
            let args = [];
            const names = [];
            const identifiers = [];
            const ctxArgs = ctx.functionCallArguments();
            if (ctxArgs.expressionList()) {
              args = ctxArgs.expressionList().expression_list().map((exprCtx) => this.visitExpression(exprCtx));
            } else if (ctxArgs.nameValueList()) {
              for (const nameValue of ctxArgs.nameValueList().nameValue_list()) {
                args.push(this.visitExpression(nameValue.expression()));
                names.push(this._toText(nameValue.identifier()));
                identifiers.push(this.visitIdentifier(nameValue.identifier()));
              }
            }
            const node = {
              type: "FunctionCall",
              expression: this.visitExpression(ctx.expression(0)),
              arguments: args,
              names,
              identifiers
            };
            return this._addMeta(node, ctx);
          }
          if (this._toText(ctx.getChild(1)) === "[" && this._toText(ctx.getChild(3)) === "]") {
            if (ctx.getChild(2).getText() === ":") {
              const node2 = {
                type: "IndexRangeAccess",
                base: this.visitExpression(ctx.expression(0))
              };
              return this._addMeta(node2, ctx);
            }
            const node = {
              type: "IndexAccess",
              base: this.visitExpression(ctx.expression(0)),
              index: this.visitExpression(ctx.expression(1))
            };
            return this._addMeta(node, ctx);
          }
          if (this._toText(ctx.getChild(1)) === "{" && this._toText(ctx.getChild(3)) === "}") {
            const node = {
              type: "NameValueExpression",
              expression: this.visitExpression(ctx.expression(0)),
              arguments: this.visitNameValueList(ctx.nameValueList())
            };
            return this._addMeta(node, ctx);
          }
          break;
        case 5:
          if (this._toText(ctx.getChild(1)) === "?" && this._toText(ctx.getChild(3)) === ":") {
            const node = {
              type: "Conditional",
              condition: this.visitExpression(ctx.expression(0)),
              trueExpression: this.visitExpression(ctx.expression(1)),
              falseExpression: this.visitExpression(ctx.expression(2))
            };
            return this._addMeta(node, ctx);
          }
          if (this._toText(ctx.getChild(1)) === "[" && this._toText(ctx.getChild(2)) === ":" && this._toText(ctx.getChild(4)) === "]") {
            const node = {
              type: "IndexRangeAccess",
              base: this.visitExpression(ctx.expression(0)),
              indexEnd: this.visitExpression(ctx.expression(1))
            };
            return this._addMeta(node, ctx);
          } else if (this._toText(ctx.getChild(1)) === "[" && this._toText(ctx.getChild(3)) === ":" && this._toText(ctx.getChild(4)) === "]") {
            const node = {
              type: "IndexRangeAccess",
              base: this.visitExpression(ctx.expression(0)),
              indexStart: this.visitExpression(ctx.expression(1))
            };
            return this._addMeta(node, ctx);
          }
          break;
        case 6:
          if (this._toText(ctx.getChild(1)) === "[" && this._toText(ctx.getChild(3)) === ":" && this._toText(ctx.getChild(5)) === "]") {
            const node = {
              type: "IndexRangeAccess",
              base: this.visitExpression(ctx.expression(0)),
              indexStart: this.visitExpression(ctx.expression(1)),
              indexEnd: this.visitExpression(ctx.expression(2))
            };
            return this._addMeta(node, ctx);
          }
          break;
      }
      throw new Error("Unrecognized expression");
    }
    visitNameValueList(ctx) {
      const names = [];
      const identifiers = [];
      const args = [];
      for (const nameValue of ctx.nameValue_list()) {
        names.push(this._toText(nameValue.identifier()));
        identifiers.push(this.visitIdentifier(nameValue.identifier()));
        args.push(this.visitExpression(nameValue.expression()));
      }
      const node = {
        type: "NameValueList",
        names,
        identifiers,
        arguments: args
      };
      return this._addMeta(node, ctx);
    }
    visitFileLevelConstant(ctx) {
      const type = this.visitTypeName(ctx.typeName());
      const name = this._toText(ctx.identifier());
      const expression = this.visitExpression(ctx.expression());
      const node = {
        type: "FileLevelConstant",
        typeName: type,
        name,
        initialValue: expression,
        isDeclaredConst: true,
        isImmutable: false
      };
      return this._addMeta(node, ctx);
    }
    visitForStatement(ctx) {
      let conditionExpression = this.visitExpressionStatement(ctx.expressionStatement());
      if (conditionExpression) {
        conditionExpression = conditionExpression.expression;
      }
      const node = {
        type: "ForStatement",
        initExpression: ctx.simpleStatement() ? this.visitSimpleStatement(ctx.simpleStatement()) : null,
        conditionExpression,
        loopExpression: {
          type: "ExpressionStatement",
          expression: ctx.expression() ? this.visitExpression(ctx.expression()) : null
        },
        body: this.visitStatement(ctx.statement())
      };
      return this._addMeta(node, ctx);
    }
    visitHexLiteral(ctx) {
      const parts = ctx.HexLiteralFragment_list().map((x2) => this._toText(x2)).map((x2) => x2.substring(4, x2.length - 1));
      const node = {
        type: "HexLiteral",
        value: parts.join(""),
        parts
      };
      return this._addMeta(node, ctx);
    }
    visitPrimaryExpression(ctx) {
      if (ctx.BooleanLiteral()) {
        const node = {
          type: "BooleanLiteral",
          value: this._toText(ctx.BooleanLiteral()) === "true"
        };
        return this._addMeta(node, ctx);
      }
      if (ctx.hexLiteral()) {
        return this.visitHexLiteral(ctx.hexLiteral());
      }
      if (ctx.stringLiteral()) {
        const fragments = ctx.stringLiteral().StringLiteralFragment_list().map((stringLiteralFragmentCtx) => {
          let text = this._toText(stringLiteralFragmentCtx);
          const isUnicode = text.slice(0, 7) === "unicode";
          if (isUnicode) {
            text = text.slice(7);
          }
          const singleQuotes = text[0] === "'";
          const textWithoutQuotes = text.substring(1, text.length - 1);
          const value = singleQuotes ? textWithoutQuotes.replace(new RegExp("\\\\'", "g"), "'") : textWithoutQuotes.replace(new RegExp('\\\\"', "g"), '"');
          return {value, isUnicode};
        });
        const parts = fragments.map((x2) => x2.value);
        const node = {
          type: "StringLiteral",
          value: parts.join(""),
          parts,
          isUnicode: fragments.map((x2) => x2.isUnicode)
        };
        return this._addMeta(node, ctx);
      }
      if (ctx.numberLiteral()) {
        return this.visitNumberLiteral(ctx.numberLiteral());
      }
      if (ctx.TypeKeyword()) {
        const node = {
          type: "Identifier",
          name: "type"
        };
        return this._addMeta(node, ctx);
      }
      if (ctx.typeName()) {
        return this.visitTypeName(ctx.typeName());
      }
      return this.visit(ctx.getChild(0));
    }
    visitTupleExpression(ctx) {
      const children = ctx.children.slice(1, -1);
      const components = this._mapCommasToNulls(children).map((expr) => {
        if (expr === null) {
          return null;
        }
        return this.visit(expr);
      });
      const node = {
        type: "TupleExpression",
        components,
        isArray: this._toText(ctx.getChild(0)) === "["
      };
      return this._addMeta(node, ctx);
    }
    buildIdentifierList(ctx) {
      const children = ctx.children.slice(1, -1);
      const identifiers = ctx.identifier_list();
      let i2 = 0;
      return this._mapCommasToNulls(children).map((identifierOrNull) => {
        if (identifierOrNull === null) {
          return null;
        }
        const iden = identifiers[i2];
        i2++;
        const node = {
          type: "VariableDeclaration",
          name: this._toText(iden),
          identifier: this.visitIdentifier(iden),
          isStateVar: false,
          isIndexed: false,
          typeName: null,
          storageLocation: null,
          expression: null
        };
        return this._addMeta(node, iden);
      });
    }
    buildVariableDeclarationList(ctx) {
      const variableDeclarations = ctx.variableDeclaration_list();
      let i2 = 0;
      return this._mapCommasToNulls(ctx.children ?? []).map((declOrNull) => {
        if (!declOrNull) {
          return null;
        }
        const decl = variableDeclarations[i2];
        i2++;
        let storageLocation = null;
        if (decl.storageLocation()) {
          storageLocation = this._toText(decl.storageLocation());
        }
        const identifierCtx = decl.identifier();
        const result = {
          type: "VariableDeclaration",
          name: this._toText(identifierCtx),
          identifier: this.visitIdentifier(identifierCtx),
          typeName: this.visitTypeName(decl.typeName()),
          storageLocation,
          isStateVar: false,
          isIndexed: false,
          expression: null
        };
        return this._addMeta(result, decl);
      });
    }
    visitImportDirective(ctx) {
      const pathString = this._toText(ctx.importPath());
      let unitAlias = null;
      let unitAliasIdentifier = null;
      let symbolAliases = null;
      let symbolAliasesIdentifiers = null;
      if (ctx.importDeclaration_list().length > 0) {
        symbolAliases = ctx.importDeclaration_list().map((decl) => {
          const symbol = this._toText(decl.identifier(0));
          let alias = null;
          if (decl.identifier_list().length > 1) {
            alias = this._toText(decl.identifier(1));
          }
          return [symbol, alias];
        });
        symbolAliasesIdentifiers = ctx.importDeclaration_list().map((decl) => {
          const symbolIdentifier = this.visitIdentifier(decl.identifier(0));
          let aliasIdentifier = null;
          if (decl.identifier_list().length > 1) {
            aliasIdentifier = this.visitIdentifier(decl.identifier(1));
          }
          return [symbolIdentifier, aliasIdentifier];
        });
      } else {
        const identifierCtxList = ctx.identifier_list();
        if (identifierCtxList.length === 0) {
        } else if (identifierCtxList.length === 1) {
          const aliasIdentifierCtx = ctx.identifier(0);
          unitAlias = this._toText(aliasIdentifierCtx);
          unitAliasIdentifier = this.visitIdentifier(aliasIdentifierCtx);
        } else if (identifierCtxList.length === 2) {
          const aliasIdentifierCtx = ctx.identifier(1);
          unitAlias = this._toText(aliasIdentifierCtx);
          unitAliasIdentifier = this.visitIdentifier(aliasIdentifierCtx);
        } else {
          throw new Error("Assertion error: an import should have one or two identifiers");
        }
      }
      const path = pathString.substring(1, pathString.length - 1);
      const pathLiteral = {
        type: "StringLiteral",
        value: path,
        parts: [path],
        isUnicode: [false]
      };
      const node = {
        type: "ImportDirective",
        path,
        pathLiteral: this._addMeta(pathLiteral, ctx.importPath()),
        unitAlias,
        unitAliasIdentifier,
        symbolAliases,
        symbolAliasesIdentifiers
      };
      return this._addMeta(node, ctx);
    }
    buildEventParameterList(ctx) {
      return ctx.eventParameter_list().map((paramCtx) => {
        const type = this.visit(paramCtx.typeName());
        const identifier = paramCtx.identifier();
        const name = identifier ? this._toText(identifier) : null;
        return {
          type: "VariableDeclaration",
          typeName: type,
          name,
          isStateVar: false,
          isIndexed: !!paramCtx.IndexedKeyword()
        };
      });
    }
    visitReturnParameters(ctx) {
      return this.visitParameterList(ctx.parameterList());
    }
    visitParameterList(ctx) {
      return ctx.parameter_list().map((paramCtx) => this.visitParameter(paramCtx));
    }
    visitInlineAssemblyStatement(ctx) {
      let language = null;
      if (ctx.StringLiteralFragment()) {
        language = this._toText(ctx.StringLiteralFragment());
        language = language.substring(1, language.length - 1);
      }
      const flags = [];
      const flag = ctx.inlineAssemblyStatementFlag();
      if (flag) {
        const flagString = this._toText(flag.stringLiteral());
        flags.push(flagString.slice(1, flagString.length - 1));
      }
      const node = {
        type: "InlineAssemblyStatement",
        language,
        flags,
        body: this.visitAssemblyBlock(ctx.assemblyBlock())
      };
      return this._addMeta(node, ctx);
    }
    visitAssemblyBlock(ctx) {
      const operations = ctx.assemblyItem_list().map((item) => this.visitAssemblyItem(item));
      const node = {
        type: "AssemblyBlock",
        operations
      };
      return this._addMeta(node, ctx);
    }
    visitAssemblyItem(ctx) {
      let text;
      if (ctx.hexLiteral()) {
        return this.visitHexLiteral(ctx.hexLiteral());
      }
      if (ctx.stringLiteral()) {
        text = this._toText(ctx.stringLiteral());
        const value = text.substring(1, text.length - 1);
        const node = {
          type: "StringLiteral",
          value,
          parts: [value],
          isUnicode: [false]
        };
        return this._addMeta(node, ctx);
      }
      if (ctx.BreakKeyword()) {
        const node = {
          type: "Break"
        };
        return this._addMeta(node, ctx);
      }
      if (ctx.ContinueKeyword()) {
        const node = {
          type: "Continue"
        };
        return this._addMeta(node, ctx);
      }
      return this.visit(ctx.getChild(0));
    }
    visitAssemblyExpression(ctx) {
      return this.visit(ctx.getChild(0));
    }
    visitAssemblyCall(ctx) {
      const functionName = this._toText(ctx.getChild(0));
      const args = ctx.assemblyExpression_list().map((assemblyExpr) => this.visitAssemblyExpression(assemblyExpr));
      const node = {
        type: "AssemblyCall",
        functionName,
        arguments: args
      };
      return this._addMeta(node, ctx);
    }
    visitAssemblyLiteral(ctx) {
      let text;
      if (ctx.stringLiteral()) {
        text = this._toText(ctx);
        const value = text.substring(1, text.length - 1);
        const node = {
          type: "StringLiteral",
          value,
          parts: [value],
          isUnicode: [false]
        };
        return this._addMeta(node, ctx);
      }
      if (ctx.BooleanLiteral()) {
        const node = {
          type: "BooleanLiteral",
          value: this._toText(ctx.BooleanLiteral()) === "true"
        };
        return this._addMeta(node, ctx);
      }
      if (ctx.DecimalNumber()) {
        const node = {
          type: "DecimalNumber",
          value: this._toText(ctx)
        };
        return this._addMeta(node, ctx);
      }
      if (ctx.HexNumber()) {
        const node = {
          type: "HexNumber",
          value: this._toText(ctx)
        };
        return this._addMeta(node, ctx);
      }
      if (ctx.hexLiteral()) {
        return this.visitHexLiteral(ctx.hexLiteral());
      }
      throw new Error("Should never reach here");
    }
    visitAssemblySwitch(ctx) {
      const node = {
        type: "AssemblySwitch",
        expression: this.visitAssemblyExpression(ctx.assemblyExpression()),
        cases: ctx.assemblyCase_list().map((c2) => this.visitAssemblyCase(c2))
      };
      return this._addMeta(node, ctx);
    }
    visitAssemblyCase(ctx) {
      let value = null;
      if (this._toText(ctx.getChild(0)) === "case") {
        value = this.visitAssemblyLiteral(ctx.assemblyLiteral());
      }
      const node = {
        type: "AssemblyCase",
        block: this.visitAssemblyBlock(ctx.assemblyBlock()),
        value,
        default: value === null
      };
      return this._addMeta(node, ctx);
    }
    visitAssemblyLocalDefinition(ctx) {
      const ctxAssemblyIdentifierOrList = ctx.assemblyIdentifierOrList();
      let names;
      if (ctxAssemblyIdentifierOrList.identifier()) {
        names = [this.visitIdentifier(ctxAssemblyIdentifierOrList.identifier())];
      } else if (ctxAssemblyIdentifierOrList.assemblyMember()) {
        names = [
          this.visitAssemblyMember(ctxAssemblyIdentifierOrList.assemblyMember())
        ];
      } else {
        names = ctxAssemblyIdentifierOrList.assemblyIdentifierList().identifier_list().map((x2) => this.visitIdentifier(x2));
      }
      let expression = null;
      if (ctx.assemblyExpression()) {
        expression = this.visitAssemblyExpression(ctx.assemblyExpression());
      }
      const node = {
        type: "AssemblyLocalDefinition",
        names,
        expression
      };
      return this._addMeta(node, ctx);
    }
    visitAssemblyFunctionDefinition(ctx) {
      const ctxAssemblyIdentifierList = ctx.assemblyIdentifierList();
      const args = ctxAssemblyIdentifierList ? ctxAssemblyIdentifierList.identifier_list().map((x2) => this.visitIdentifier(x2)) : [];
      const ctxAssemblyFunctionReturns = ctx.assemblyFunctionReturns();
      const returnArgs = ctxAssemblyFunctionReturns ? ctxAssemblyFunctionReturns.assemblyIdentifierList().identifier_list().map((x2) => this.visitIdentifier(x2)) : [];
      const node = {
        type: "AssemblyFunctionDefinition",
        name: this._toText(ctx.identifier()),
        arguments: args,
        returnArguments: returnArgs,
        body: this.visitAssemblyBlock(ctx.assemblyBlock())
      };
      return this._addMeta(node, ctx);
    }
    visitAssemblyAssignment(ctx) {
      const ctxAssemblyIdentifierOrList = ctx.assemblyIdentifierOrList();
      let names;
      if (ctxAssemblyIdentifierOrList.identifier()) {
        names = [this.visitIdentifier(ctxAssemblyIdentifierOrList.identifier())];
      } else if (ctxAssemblyIdentifierOrList.assemblyMember()) {
        names = [
          this.visitAssemblyMember(ctxAssemblyIdentifierOrList.assemblyMember())
        ];
      } else {
        names = ctxAssemblyIdentifierOrList.assemblyIdentifierList().identifier_list().map((x2) => this.visitIdentifier(x2));
      }
      const node = {
        type: "AssemblyAssignment",
        names,
        expression: this.visitAssemblyExpression(ctx.assemblyExpression())
      };
      return this._addMeta(node, ctx);
    }
    visitAssemblyMember(ctx) {
      const [accessed, member] = ctx.identifier_list();
      const node = {
        type: "AssemblyMemberAccess",
        expression: this.visitIdentifier(accessed),
        memberName: this.visitIdentifier(member)
      };
      return this._addMeta(node, ctx);
    }
    visitLabelDefinition(ctx) {
      const node = {
        type: "LabelDefinition",
        name: this._toText(ctx.identifier())
      };
      return this._addMeta(node, ctx);
    }
    visitAssemblyStackAssignment(ctx) {
      const node = {
        type: "AssemblyStackAssignment",
        name: this._toText(ctx.identifier()),
        expression: this.visitAssemblyExpression(ctx.assemblyExpression())
      };
      return this._addMeta(node, ctx);
    }
    visitAssemblyFor(ctx) {
      const node = {
        type: "AssemblyFor",
        pre: this.visit(ctx.getChild(1)),
        condition: this.visit(ctx.getChild(2)),
        post: this.visit(ctx.getChild(3)),
        body: this.visit(ctx.getChild(4))
      };
      return this._addMeta(node, ctx);
    }
    visitAssemblyIf(ctx) {
      const node = {
        type: "AssemblyIf",
        condition: this.visitAssemblyExpression(ctx.assemblyExpression()),
        body: this.visitAssemblyBlock(ctx.assemblyBlock())
      };
      return this._addMeta(node, ctx);
    }
    visitContinueStatement(ctx) {
      const node = {
        type: "ContinueStatement"
      };
      return this._addMeta(node, ctx);
    }
    visitBreakStatement(ctx) {
      const node = {
        type: "BreakStatement"
      };
      return this._addMeta(node, ctx);
    }
    _toText(ctx) {
      const text = ctx.getText();
      if (text === void 0 || text === null) {
        throw new Error("Assertion error: text should never be undefined");
      }
      return text;
    }
    _stateMutabilityToText(ctx) {
      if (ctx.PureKeyword()) {
        return "pure";
      }
      if (ctx.ConstantKeyword()) {
        return "constant";
      }
      if (ctx.PayableKeyword()) {
        return "payable";
      }
      if (ctx.ViewKeyword()) {
        return "view";
      }
      throw new Error("Assertion error: non-exhaustive stateMutability check");
    }
    _loc(ctx) {
      const sourceLocation = {
        start: {
          line: ctx.start.line,
          column: ctx.start.column
        },
        end: {
          line: ctx.stop ? ctx.stop.line : ctx.start.line,
          column: ctx.stop ? ctx.stop.column : ctx.start.column
        }
      };
      return sourceLocation;
    }
    _range(ctx) {
      return [ctx.start.start, ctx.stop?.stop ?? ctx.start.start];
    }
    _addMeta(node, ctx) {
      const nodeWithMeta = {
        type: node.type
      };
      if (this.options.loc === true) {
        node.loc = this._loc(ctx);
      }
      if (this.options.range === true) {
        node.range = this._range(ctx);
      }
      return {
        ...nodeWithMeta,
        ...node
      };
    }
    _mapCommasToNulls(children) {
      if (children.length === 0) {
        return [];
      }
      const values = [];
      let comma = true;
      for (const el of children) {
        if (comma) {
          if (this._toText(el) === ",") {
            values.push(null);
          } else {
            values.push(el);
            comma = false;
          }
        } else {
          if (this._toText(el) !== ",") {
            throw new Error("expected comma");
          }
          comma = true;
        }
      }
      if (comma) {
        values.push(null);
      }
      return values;
    }
  };
  function isBinOp(op) {
    return binaryOpValues.includes(op);
  }

  // src/ErrorListener.ts
  var ErrorListener = class extends g {
    constructor() {
      super();
      this._errors = [];
    }
    syntaxError(recognizer, offendingSymbol, line, column, message) {
      this._errors.push({message, line, column});
    }
    getErrors() {
      return this._errors;
    }
    hasErrors() {
      return this._errors.length > 0;
    }
  };
  var ErrorListener_default = ErrorListener;

  // src/antlr/solidity-tokens.ts
  var tokens = {
    "1": "pragma",
    "2": ";",
    "3": "*",
    "4": "||",
    "5": "^",
    "6": "~",
    "7": ">=",
    "8": ">",
    "9": "<",
    "10": "<=",
    "11": "=",
    "12": "as",
    "13": "import",
    "14": "from",
    "15": "{",
    "16": ",",
    "17": "}",
    "18": "abstract",
    "19": "contract",
    "20": "interface",
    "21": "library",
    "22": "is",
    "23": "(",
    "24": ")",
    "25": "error",
    "26": "using",
    "27": "for",
    "28": "|",
    "29": "&",
    "30": "+",
    "31": "-",
    "32": "/",
    "33": "%",
    "34": "==",
    "35": "!=",
    "36": "struct",
    "37": "modifier",
    "38": "function",
    "39": "returns",
    "40": "event",
    "41": "enum",
    "42": "[",
    "43": "]",
    "44": "address",
    "45": ".",
    "46": "mapping",
    "47": "=>",
    "48": "memory",
    "49": "storage",
    "50": "calldata",
    "51": "if",
    "52": "else",
    "53": "try",
    "54": "catch",
    "55": "while",
    "56": "unchecked",
    "57": "assembly",
    "58": "do",
    "59": "return",
    "60": "throw",
    "61": "emit",
    "62": "revert",
    "63": "var",
    "64": "bool",
    "65": "string",
    "66": "byte",
    "67": "++",
    "68": "--",
    "69": "new",
    "70": ":",
    "71": "delete",
    "72": "!",
    "73": "**",
    "74": "<<",
    "75": ">>",
    "76": "&&",
    "77": "?",
    "78": "|=",
    "79": "^=",
    "80": "&=",
    "81": "<<=",
    "82": ">>=",
    "83": "+=",
    "84": "-=",
    "85": "*=",
    "86": "/=",
    "87": "%=",
    "88": "let",
    "89": ":=",
    "90": "=:",
    "91": "switch",
    "92": "case",
    "93": "default",
    "94": "->",
    "95": "callback",
    "96": "override",
    "97": "Int",
    "98": "Uint",
    "99": "Byte",
    "100": "Fixed",
    "101": "Ufixed",
    "102": "BooleanLiteral",
    "103": "DecimalNumber",
    "104": "HexNumber",
    "105": "NumberUnit",
    "106": "HexLiteralFragment",
    "107": "ReservedKeyword",
    "108": "anonymous",
    "109": "break",
    "110": "constant",
    "111": "immutable",
    "112": "continue",
    "113": "leave",
    "114": "external",
    "115": "indexed",
    "116": "internal",
    "117": "payable",
    "118": "private",
    "119": "public",
    "120": "virtual",
    "121": "pure",
    "122": "type",
    "123": "view",
    "124": "global",
    "125": "constructor",
    "126": "fallback",
    "127": "receive",
    "128": "Identifier",
    "129": "StringLiteralFragment",
    "130": "VersionLiteral",
    "131": "WS",
    "132": "COMMENT",
    "133": "LINE_COMMENT"
  };

  // src/tokens.ts
  var TYPE_TOKENS = [
    "var",
    "bool",
    "address",
    "string",
    "Int",
    "Uint",
    "Byte",
    "Fixed",
    "UFixed"
  ];
  function getTokenType(value) {
    if (value === "Identifier" || value === "from") {
      return "Identifier";
    } else if (value === "TrueLiteral" || value === "FalseLiteral") {
      return "Boolean";
    } else if (value === "VersionLiteral") {
      return "Version";
    } else if (value === "StringLiteral") {
      return "String";
    } else if (TYPE_TOKENS.includes(value)) {
      return "Type";
    } else if (value === "NumberUnit") {
      return "Subdenomination";
    } else if (value === "DecimalNumber") {
      return "Numeric";
    } else if (value === "HexLiteral") {
      return "Hex";
    } else if (value === "ReservedKeyword") {
      return "Reserved";
    } else if (/^\W+$/.test(value)) {
      return "Punctuator";
    } else {
      return "Keyword";
    }
  }
  function range(token) {
    return [token.start, token.stop + 1];
  }
  function loc(token) {
    const tokenText = token.text ?? "";
    const textInLines = tokenText.split(/\r?\n/);
    const numberOfNewLines = textInLines.length - 1;
    return {
      start: {line: token.line, column: token.column},
      end: {
        line: token.line + numberOfNewLines,
        column: textInLines[numberOfNewLines].length + (numberOfNewLines === 0 ? token.column : 0)
      }
    };
  }
  function buildTokenList(tokensArg, options) {
    return tokensArg.map((token) => {
      const type = getTokenType(tokens[token.type.toString()]);
      const node = {type, value: token.text};
      if (options.range === true) {
        node.range = range(token);
      }
      if (options.loc === true) {
        node.loc = loc(token);
      }
      return node;
    });
  }
  function buildCommentList(tokensArg, commentsChannelId, options) {
    return tokensArg.filter((token) => token.channel === commentsChannelId).map((token) => {
      const comment = token.text.startsWith("//") ? {type: "LineComment", value: token.text.slice(2)} : {type: "BlockComment", value: token.text.slice(2, -2)};
      if (options.range === true) {
        comment.range = range(token);
      }
      if (options.loc === true) {
        comment.loc = loc(token);
      }
      return comment;
    });
  }

  // src/parser.ts
  var ParserError = class extends Error {
    constructor(args) {
      super();
      const {message, line, column} = args.errors[0];
      this.message = `${message} (${line}:${column})`;
      this.errors = args.errors;
      if (Error.captureStackTrace !== void 0) {
        Error.captureStackTrace(this, this.constructor);
      } else {
        this.stack = new Error().stack;
      }
    }
  };
  function tokenize(input, options = {}) {
    const inputStream = new a(input);
    const lexer = new SolidityLexer_default(inputStream);
    return buildTokenList(lexer.getAllTokens(), options);
  }
  function parse(input, options = {}) {
    const inputStream = new a(input);
    const lexer = new SolidityLexer_default(inputStream);
    const tokenStream = new c(lexer);
    const parser = new SolidityParser_default(tokenStream);
    const listener = new ErrorListener_default();
    lexer.removeErrorListeners();
    lexer.addErrorListener(listener);
    parser.removeErrorListeners();
    parser.addErrorListener(listener);
    parser.buildParseTrees = true;
    const sourceUnit = parser.sourceUnit();
    const astBuilder = new ASTBuilder(options);
    astBuilder.visit(sourceUnit);
    const ast = astBuilder.result;
    if (ast === null) {
      throw new Error("ast should never be null");
    }
    if (options.tokens === true) {
      ast.tokens = buildTokenList(tokenStream.tokens, options);
    }
    if (options.comments === true) {
      ast.comments = buildCommentList(tokenStream.tokens, lexer.channelNames.indexOf("HIDDEN"), options);
    }
    if (listener.hasErrors()) {
      if (options.tolerant !== true) {
        throw new ParserError({errors: listener.getErrors()});
      }
      ast.errors = listener.getErrors();
    }
    return ast;
  }
  function _isASTNode(node) {
    if (typeof node !== "object" || node === null) {
      return false;
    }
    const nodeAsASTNode = node;
    if (Object.prototype.hasOwnProperty.call(nodeAsASTNode, "type") && typeof nodeAsASTNode.type === "string") {
      return astNodeTypes.includes(nodeAsASTNode.type);
    }
    return false;
  }
  function visit(node, visitor, nodeParent) {
    if (Array.isArray(node)) {
      node.forEach((child) => visit(child, visitor, nodeParent));
    }
    if (!_isASTNode(node))
      return;
    let cont = true;
    if (visitor[node.type] !== void 0) {
      cont = visitor[node.type](node, nodeParent);
    }
    if (cont === false)
      return;
    for (const prop in node) {
      if (Object.prototype.hasOwnProperty.call(node, prop)) {
        visit(node[prop], visitor, node);
      }
    }
    const selector = node.type + ":exit";
    if (visitor[selector] !== void 0) {
      visitor[selector](node, nodeParent);
    }
  }
  return src_exports;
})();

  return SolidityParser;
});
//# sourceMappingURL=index.umd.js.map
