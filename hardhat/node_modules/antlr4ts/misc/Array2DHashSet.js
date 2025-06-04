"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
var __decorate = (this && this.__decorate) || function (decorators, target, key, desc) {
    var c = arguments.length, r = c < 3 ? target : desc === null ? desc = Object.getOwnPropertyDescriptor(target, key) : desc, d;
    if (typeof Reflect === "object" && typeof Reflect.decorate === "function") r = Reflect.decorate(decorators, target, key, desc);
    else for (var i = decorators.length - 1; i >= 0; i--) if (d = decorators[i]) r = (c < 3 ? d(r) : c > 3 ? d(target, key, r) : d(target, key)) || r;
    return c > 3 && r && Object.defineProperty(target, key, r), r;
};
var __param = (this && this.__param) || function (paramIndex, decorator) {
    return function (target, key) { decorator(target, key, paramIndex); }
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Array2DHashSet = void 0;
// ConvertTo-TS run at 2016-10-03T02:09:41.7434086-07:00
const assert = require("assert");
const DefaultEqualityComparator_1 = require("./DefaultEqualityComparator");
const Decorators_1 = require("../Decorators");
const MurmurHash_1 = require("./MurmurHash");
/** {@link Set} implementation with closed hashing (open addressing). */
// NOTE:  JavaScript's Set interface has on significant different diffrence from Java's:
// 		  e.g. the return type of add() differs!
//        For this reason I've commented tweaked the implements clause
const INITAL_CAPACITY = 16; // must be power of 2
const LOAD_FACTOR = 0.75;
class Array2DHashSet {
    constructor(comparatorOrSet, initialCapacity = INITAL_CAPACITY) {
        /** How many elements in set */
        this.n = 0;
        this.threshold = Math.floor(INITAL_CAPACITY * LOAD_FACTOR); // when to expand
        if (comparatorOrSet instanceof Array2DHashSet) {
            this.comparator = comparatorOrSet.comparator;
            this.buckets = comparatorOrSet.buckets.slice(0);
            for (let i = 0; i < this.buckets.length; i++) {
                let bucket = this.buckets[i];
                if (bucket) {
                    this.buckets[i] = bucket.slice(0);
                }
            }
            this.n = comparatorOrSet.n;
            this.threshold = comparatorOrSet.threshold;
        }
        else {
            this.comparator = comparatorOrSet || DefaultEqualityComparator_1.DefaultEqualityComparator.INSTANCE;
            this.buckets = this.createBuckets(initialCapacity);
        }
    }
    /**
     * Add `o` to set if not there; return existing value if already
     * there. This method performs the same operation as {@link #add} aside from
     * the return value.
     */
    getOrAdd(o) {
        if (this.n > this.threshold) {
            this.expand();
        }
        return this.getOrAddImpl(o);
    }
    getOrAddImpl(o) {
        let b = this.getBucket(o);
        let bucket = this.buckets[b];
        // NEW BUCKET
        if (!bucket) {
            bucket = [o];
            this.buckets[b] = bucket;
            this.n++;
            return o;
        }
        // LOOK FOR IT IN BUCKET
        for (let existing of bucket) {
            if (this.comparator.equals(existing, o)) {
                return existing; // found existing, quit
            }
        }
        // FULL BUCKET, expand and add to end
        bucket.push(o);
        this.n++;
        return o;
    }
    get(o) {
        if (o == null) {
            return o;
        }
        let b = this.getBucket(o);
        let bucket = this.buckets[b];
        if (!bucket) {
            // no bucket
            return undefined;
        }
        for (let e of bucket) {
            if (this.comparator.equals(e, o)) {
                return e;
            }
        }
        return undefined;
    }
    getBucket(o) {
        let hash = this.comparator.hashCode(o);
        let b = hash & (this.buckets.length - 1); // assumes len is power of 2
        return b;
    }
    hashCode() {
        let hash = MurmurHash_1.MurmurHash.initialize();
        for (let bucket of this.buckets) {
            if (bucket == null) {
                continue;
            }
            for (let o of bucket) {
                if (o == null) {
                    break;
                }
                hash = MurmurHash_1.MurmurHash.update(hash, this.comparator.hashCode(o));
            }
        }
        hash = MurmurHash_1.MurmurHash.finish(hash, this.size);
        return hash;
    }
    equals(o) {
        if (o === this) {
            return true;
        }
        if (!(o instanceof Array2DHashSet)) {
            return false;
        }
        if (o.size !== this.size) {
            return false;
        }
        let same = this.containsAll(o);
        return same;
    }
    expand() {
        let old = this.buckets;
        let newCapacity = this.buckets.length * 2;
        let newTable = this.createBuckets(newCapacity);
        this.buckets = newTable;
        this.threshold = Math.floor(newCapacity * LOAD_FACTOR);
        //		System.out.println("new size="+newCapacity+", thres="+threshold);
        // rehash all existing entries
        let oldSize = this.size;
        for (let bucket of old) {
            if (!bucket) {
                continue;
            }
            for (let o of bucket) {
                let b = this.getBucket(o);
                let newBucket = this.buckets[b];
                if (!newBucket) {
                    newBucket = [];
                    this.buckets[b] = newBucket;
                }
                newBucket.push(o);
            }
        }
        assert(this.n === oldSize);
    }
    add(t) {
        let existing = this.getOrAdd(t);
        return existing === t;
    }
    get size() {
        return this.n;
    }
    get isEmpty() {
        return this.n === 0;
    }
    contains(o) {
        return this.containsFast(this.asElementType(o));
    }
    containsFast(obj) {
        if (obj == null) {
            return false;
        }
        return this.get(obj) != null;
    }
    *[Symbol.iterator]() {
        yield* this.toArray();
    }
    toArray() {
        const a = new Array(this.size);
        // Copy elements from the nested arrays into the destination array
        let i = 0; // Position within destination array
        for (let bucket of this.buckets) {
            if (bucket == null) {
                continue;
            }
            for (let o of bucket) {
                if (o == null) {
                    break;
                }
                a[i++] = o;
            }
        }
        return a;
    }
    containsAll(collection) {
        if (collection instanceof Array2DHashSet) {
            let s = collection;
            for (let bucket of s.buckets) {
                if (bucket == null) {
                    continue;
                }
                for (let o of bucket) {
                    if (o == null) {
                        break;
                    }
                    if (!this.containsFast(this.asElementType(o))) {
                        return false;
                    }
                }
            }
        }
        else {
            for (let o of collection) {
                if (!this.containsFast(this.asElementType(o))) {
                    return false;
                }
            }
        }
        return true;
    }
    addAll(c) {
        let changed = false;
        for (let o of c) {
            let existing = this.getOrAdd(o);
            if (existing !== o) {
                changed = true;
            }
        }
        return changed;
    }
    clear() {
        this.buckets = this.createBuckets(INITAL_CAPACITY);
        this.n = 0;
        this.threshold = Math.floor(INITAL_CAPACITY * LOAD_FACTOR);
    }
    toString() {
        if (this.size === 0) {
            return "{}";
        }
        let buf = "{";
        let first = true;
        for (let bucket of this.buckets) {
            if (bucket == null) {
                continue;
            }
            for (let o of bucket) {
                if (o == null) {
                    break;
                }
                if (first) {
                    first = false;
                }
                else {
                    buf += ", ";
                }
                buf += o.toString();
            }
        }
        buf += "}";
        return buf;
    }
    toTableString() {
        let buf = "";
        for (let bucket of this.buckets) {
            if (bucket == null) {
                buf += "null\n";
                continue;
            }
            buf += "[";
            let first = true;
            for (let o of bucket) {
                if (first) {
                    first = false;
                }
                else {
                    buf += " ";
                }
                if (o == null) {
                    buf += "_";
                }
                else {
                    buf += o.toString();
                }
            }
            buf += "]\n";
        }
        return buf;
    }
    /**
     * Return `o` as an instance of the element type `T`. If
     * `o` is non-undefined but known to not be an instance of `T`, this
     * method returns `undefined`. The base implementation does not perform any
     * type checks; override this method to provide strong type checks for the
     * {@link #contains} and {@link #remove} methods to ensure the arguments to
     * the {@link EqualityComparator} for the set always have the expected
     * types.
     *
     * @param o the object to try and cast to the element type of the set
     * @returns `o` if it could be an instance of `T`, otherwise
     * `undefined`.
     */
    asElementType(o) {
        return o;
    }
    /**
     * Return an array of `T[]` with length `capacity`.
     *
     * @param capacity the length of the array to return
     * @returns the newly constructed array
     */
    createBuckets(capacity) {
        return new Array(capacity);
    }
}
__decorate([
    Decorators_1.NotNull
], Array2DHashSet.prototype, "comparator", void 0);
__decorate([
    Decorators_1.Override
], Array2DHashSet.prototype, "hashCode", null);
__decorate([
    Decorators_1.Override
], Array2DHashSet.prototype, "equals", null);
__decorate([
    Decorators_1.Override
], Array2DHashSet.prototype, "add", null);
__decorate([
    Decorators_1.Override
], Array2DHashSet.prototype, "size", null);
__decorate([
    Decorators_1.Override
], Array2DHashSet.prototype, "isEmpty", null);
__decorate([
    Decorators_1.Override
], Array2DHashSet.prototype, "contains", null);
__decorate([
    __param(0, Decorators_1.Nullable)
], Array2DHashSet.prototype, "containsFast", null);
__decorate([
    Decorators_1.Override
], Array2DHashSet.prototype, Symbol.iterator, null);
__decorate([
    Decorators_1.Override
], Array2DHashSet.prototype, "toArray", null);
__decorate([
    Decorators_1.Override
], Array2DHashSet.prototype, "containsAll", null);
__decorate([
    Decorators_1.Override
], Array2DHashSet.prototype, "addAll", null);
__decorate([
    Decorators_1.Override
], Array2DHashSet.prototype, "clear", null);
__decorate([
    Decorators_1.Override
], Array2DHashSet.prototype, "toString", null);
__decorate([
    Decorators_1.SuppressWarnings("unchecked")
], Array2DHashSet.prototype, "asElementType", null);
__decorate([
    Decorators_1.SuppressWarnings("unchecked")
], Array2DHashSet.prototype, "createBuckets", null);
exports.Array2DHashSet = Array2DHashSet;
//# sourceMappingURL=Array2DHashSet.js.map