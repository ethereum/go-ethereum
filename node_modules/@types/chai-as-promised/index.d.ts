/// <reference types="chai" />

declare module "chai-as-promised" {
    interface ChaiAsPromised extends Chai.ChaiPlugin {
        transferPromiseness(assertion: Chai.PromisedAssertion, promise: PromiseLike<any>): void;
        transformAsserterArgs(values: any[]): any;
    }

    const chaiAsPromised: ChaiAsPromised;
    export = chaiAsPromised;
}

declare namespace Chai {
    // For BDD API
    interface Assertion extends LanguageChains, NumericComparison, TypeComparison {
        eventually: PromisedAssertion;
        become(expected: any): PromisedAssertion;
        fulfilled: PromisedAssertion;
        rejected: PromisedAssertion;
        rejectedWith: PromisedThrow;
        notify(fn: Function): PromisedAssertion;
    }

    // Eventually does not have .then(), but PromisedAssertion have.
    interface Eventually extends PromisedLanguageChains, PromisedNumericComparison, PromisedTypeComparison {
        // From chai-as-promised
        become(expected: any): PromisedAssertion;
        fulfilled: PromisedAssertion;
        rejected: PromisedAssertion;
        rejectedWith: PromisedThrow;
        notify(fn: Function): PromisedAssertion;

        // From chai
        not: PromisedAssertion;
        deep: PromisedDeep;
        ordered: PromisedOrdered;
        nested: PromisedNested;
        any: PromisedKeyFilter;
        all: PromisedKeyFilter;
        a: PromisedTypeComparison;
        an: PromisedTypeComparison;
        include: PromisedInclude;
        contain: PromisedInclude;
        ok: PromisedAssertion;
        true: PromisedAssertion;
        false: PromisedAssertion;
        null: PromisedAssertion;
        undefined: PromisedAssertion;
        NaN: PromisedAssertion;
        exist: PromisedAssertion;
        empty: PromisedAssertion;
        arguments: PromisedAssertion;
        Arguments: PromisedAssertion;
        equal: PromisedEqual;
        equals: PromisedEqual;
        eq: PromisedEqual;
        eql: PromisedEqual;
        eqls: PromisedEqual;
        property: PromisedProperty;
        ownProperty: PromisedOwnProperty;
        haveOwnProperty: PromisedOwnProperty;
        ownPropertyDescriptor: PromisedOwnPropertyDescriptor;
        haveOwnPropertyDescriptor: PromisedOwnPropertyDescriptor;
        length: PromisedLength;
        lengthOf: PromisedLength;
        match: PromisedMatch;
        matches: PromisedMatch;
        string(string: string, message?: string): PromisedAssertion;
        keys: PromisedKeys;
        key(string: string): PromisedAssertion;
        throw: PromisedThrow;
        throws: PromisedThrow;
        Throw: PromisedThrow;
        respondTo: PromisedRespondTo;
        respondsTo: PromisedRespondTo;
        itself: PromisedAssertion;
        satisfy: PromisedSatisfy;
        satisfies: PromisedSatisfy;
        closeTo: PromisedCloseTo;
        approximately: PromisedCloseTo;
        members: PromisedMembers;
        increase: PromisedPropertyChange;
        increases: PromisedPropertyChange;
        decrease: PromisedPropertyChange;
        decreases: PromisedPropertyChange;
        change: PromisedPropertyChange;
        changes: PromisedPropertyChange;
        extensible: PromisedAssertion;
        sealed: PromisedAssertion;
        frozen: PromisedAssertion;
        oneOf(list: any[], message?: string): PromisedAssertion;
    }

    interface PromisedAssertion extends Eventually, PromiseLike<any> {
    }

    interface PromisedLanguageChains {
        eventually: Eventually;

        // From chai
        to: PromisedAssertion;
        be: PromisedAssertion;
        been: PromisedAssertion;
        is: PromisedAssertion;
        that: PromisedAssertion;
        which: PromisedAssertion;
        and: PromisedAssertion;
        has: PromisedAssertion;
        have: PromisedAssertion;
        with: PromisedAssertion;
        at: PromisedAssertion;
        of: PromisedAssertion;
        same: PromisedAssertion;
        but: PromisedAssertion;
        does: PromisedAssertion;
    }

    interface PromisedNumericComparison {
        above: PromisedNumberComparer;
        gt: PromisedNumberComparer;
        greaterThan: PromisedNumberComparer;
        least: PromisedNumberComparer;
        gte: PromisedNumberComparer;
        below: PromisedNumberComparer;
        lt: PromisedNumberComparer;
        lessThan: PromisedNumberComparer;
        most: PromisedNumberComparer;
        lte: PromisedNumberComparer;
        within(start: number, finish: number, message?: string): PromisedAssertion;
    }

    interface PromisedNumberComparer {
        (value: number, message?: string): PromisedAssertion;
    }

    interface PromisedTypeComparison {
        (type: string, message?: string): PromisedAssertion;
        instanceof: PromisedInstanceOf;
        instanceOf: PromisedInstanceOf;
    }

    interface PromisedInstanceOf {
        (constructor: Object, message?: string): PromisedAssertion;
    }

    interface PromisedCloseTo {
        (expected: number, delta: number, message?: string): PromisedAssertion;
    }

    interface PromisedNested {
        include: PromisedInclude;
        property: PromisedProperty;
        members: PromisedMembers;
    }

    interface PromisedDeep {
        equal: PromisedEqual;
        equals: PromisedEqual;
        eq: PromisedEqual;
        include: PromisedInclude;
        property: PromisedProperty;
        members: PromisedMembers;
        ordered: PromisedOrdered;
    }

    interface PromisedOrdered {
        members: PromisedMembers;
    }

    interface PromisedKeyFilter {
        keys: PromisedKeys;
    }

    interface PromisedEqual {
        (value: any, message?: string): PromisedAssertion;
    }

    interface PromisedProperty {
        (name: string | symbol, value?: any, message?: string): PromisedAssertion;
    }

    interface PromisedOwnProperty {
        (name: string | symbol, message?: string): PromisedAssertion;
    }

    interface PromisedOwnPropertyDescriptor {
        (name: string | symbol, descriptor: PropertyDescriptor, message?: string): PromisedAssertion;
        (name: string | symbol, message?: string): PromisedAssertion;
    }

    interface PromisedLength extends PromisedLanguageChains, PromisedNumericComparison {
        (length: number, message?: string): PromisedAssertion;
    }

    interface PromisedInclude {
        (value: Object, message?: string): PromisedAssertion;
        (value: string, message?: string): PromisedAssertion;
        (value: number, message?: string): PromisedAssertion;
        keys: PromisedKeys;
        deep: PromisedDeep;
        ordered: PromisedOrdered;
        members: PromisedMembers;
        any: PromisedKeyFilter;
        all: PromisedKeyFilter;
    }

    interface PromisedMatch {
        (regexp: RegExp | string, message?: string): PromisedAssertion;
    }

    interface PromisedKeys {
        (...keys: string[]): PromisedAssertion;
        (keys: any[]): PromisedAssertion;
        (keys: Object): PromisedAssertion;
    }

    interface PromisedThrow {
        (): PromisedAssertion;
        (expected: string | RegExp, message?: string): PromisedAssertion;
        (constructor: Error | Function, expected?: string | RegExp, message?: string): PromisedAssertion;
    }

    interface PromisedRespondTo {
        (method: string, message?: string): PromisedAssertion;
    }

    interface PromisedSatisfy {
        (matcher: Function, message?: string): PromisedAssertion;
    }

    interface PromisedMembers {
        (set: any[], message?: string): PromisedAssertion;
    }

    interface PromisedPropertyChange {
        (object: Object, property: string, message?: string): PromisedAssertion;
    }

    // For Assert API
    interface Assert {
        eventually: PromisedAssert;
        isFulfilled(promise: PromiseLike<any>, message?: string): PromiseLike<void>;
        becomes(promise: PromiseLike<any>, expected: any, message?: string): PromiseLike<void>;
        doesNotBecome(promise: PromiseLike<any>, expected: any, message?: string): PromiseLike<void>;
        isRejected(promise: PromiseLike<any>, message?: string): PromiseLike<void>;
        isRejected(promise: PromiseLike<any>, expected: any, message?: string): PromiseLike<void>;
        isRejected(promise: PromiseLike<any>, match: RegExp, message?: string): PromiseLike<void>;
        notify(fn: Function): PromiseLike<void>;
    }

    export interface PromisedAssert {
        fail(actual?: PromiseLike<any>, expected?: any, msg?: string, operator?: string): PromiseLike<void>;

        isOk(val: PromiseLike<any>, msg?: string): PromiseLike<void>;
        ok(val: PromiseLike<any>, msg?: string): PromiseLike<void>;
        isNotOk(val: PromiseLike<any>, msg?: string): PromiseLike<void>;
        notOk(val: PromiseLike<any>, msg?: string): PromiseLike<void>;

        equal(act: PromiseLike<any>, exp: any, msg?: string): PromiseLike<void>;
        notEqual(act: PromiseLike<any>, exp: any, msg?: string): PromiseLike<void>;

        strictEqual(act: PromiseLike<any>, exp: any, msg?: string): PromiseLike<void>;
        notStrictEqual(act: PromiseLike<any>, exp: any, msg?: string): PromiseLike<void>;

        deepEqual(act: PromiseLike<any>, exp: any, msg?: string): PromiseLike<void>;
        notDeepEqual(act: PromiseLike<any>, exp: any, msg?: string): PromiseLike<void>;

        isAbove(val: PromiseLike<number>, above: number, msg?: string): PromiseLike<void>;
        isAtLeast(val: PromiseLike<number>, atLeast: number, msg?: string): PromiseLike<void>;
        isAtBelow(val: PromiseLike<number>, below: number, msg?: string): PromiseLike<void>;
        isAtMost(val: PromiseLike<number>, atMost: number, msg?: string): PromiseLike<void>;

        isTrue(val: PromiseLike<any>, msg?: string): PromiseLike<void>;
        isFalse(val: PromiseLike<any>, msg?: string): PromiseLike<void>;

        isNotTrue(val: PromiseLike<any>, msg?: string): PromiseLike<void>;
        isNotFalse(val: PromiseLike<any>, msg?: string): PromiseLike<void>;

        isNull(val: PromiseLike<any>, msg?: string): PromiseLike<void>;
        isNotNull(val: PromiseLike<any>, msg?: string): PromiseLike<void>;

        isNaN(val: PromiseLike<any>, msg?: string): PromiseLike<void>;
        isNotNaN(val: PromiseLike<any>, msg?: string): PromiseLike<void>;

        exists(val: PromiseLike<any>, msg?: string): PromiseLike<void>;
        notExists(val: PromiseLike<any>, msg?: string): PromiseLike<void>;

        isUndefined(val: PromiseLike<any>, msg?: string): PromiseLike<void>;
        isDefined(val: PromiseLike<any>, msg?: string): PromiseLike<void>;

        isFunction(val: PromiseLike<any>, msg?: string): PromiseLike<void>;
        isNotFunction(val: PromiseLike<any>, msg?: string): PromiseLike<void>;

        isObject(val: PromiseLike<any>, msg?: string): PromiseLike<void>;
        isNotObject(val: PromiseLike<any>, msg?: string): PromiseLike<void>;

        isArray(val: PromiseLike<any>, msg?: string): PromiseLike<void>;
        isNotArray(val: PromiseLike<any>, msg?: string): PromiseLike<void>;

        isString(val: PromiseLike<any>, msg?: string): PromiseLike<void>;
        isNotString(val: PromiseLike<any>, msg?: string): PromiseLike<void>;

        isNumber(val: PromiseLike<any>, msg?: string): PromiseLike<void>;
        isNotNumber(val: PromiseLike<any>, msg?: string): PromiseLike<void>;
        isFinite(val: PromiseLike<number>, msg?: string): PromiseLike<void>;

        isBoolean(val: PromiseLike<any>, msg?: string): PromiseLike<void>;
        isNotBoolean(val: PromiseLike<any>, msg?: string): PromiseLike<void>;

        typeOf(val: PromiseLike<any>, type: string, msg?: string): PromiseLike<void>;
        notTypeOf(val: PromiseLike<any>, type: string, msg?: string): PromiseLike<void>;

        instanceOf(val: PromiseLike<any>, type: Function, msg?: string): PromiseLike<void>;
        notInstanceOf(val: PromiseLike<any>, type: Function, msg?: string): PromiseLike<void>;

        include(exp: PromiseLike<string>, inc: any, msg?: string): PromiseLike<void>;
        include(exp: PromiseLike<any[]>, inc: any, msg?: string): PromiseLike<void>;

        notInclude(exp: PromiseLike<string>, inc: any, msg?: string): PromiseLike<void>;
        notInclude(exp: PromiseLike<any[]>, inc: any, msg?: string): PromiseLike<void>;

        deepInclude(exp: PromiseLike<string>, inc: any, msg?: string): PromiseLike<void>;
        deepInclude(exp: PromiseLike<any[]>, inc: any, msg?: string): PromiseLike<void>;

        notDeepInclude(exp: PromiseLike<string>, inc: any, msg?: string): PromiseLike<void>;
        notDeepInclude(exp: PromiseLike<any[]>, inc: any, msg?: string): PromiseLike<void>;

        nestedInclude(exp: PromiseLike<Object>, inc: Object, msg?: string): PromiseLike<void>;
        notNestedInclude(exp: PromiseLike<Object>, inc: Object, msg?: string): PromiseLike<void>;

        deepNestedInclude(exp: PromiseLike<Object>, inc: Object, msg?: string): PromiseLike<void>;
        notDeepNestedInclude(exp: PromiseLike<Object>, inc: Object, msg?: string): PromiseLike<void>;

        ownInclude(exp: PromiseLike<Object>, inc: Object, msg?: string): PromiseLike<void>;
        notOwnInclude(exp: PromiseLike<Object>, inc: Object, msg?: string): PromiseLike<void>;

        deepOwnInclude(exp: PromiseLike<Object>, inc: Object, msg?: string): PromiseLike<void>;
        notDeepOwnInclude(exp: PromiseLike<Object>, inc: Object, msg?: string): PromiseLike<void>;

        match(exp: PromiseLike<any>, re: RegExp, msg?: string): PromiseLike<void>;
        notMatch(exp: PromiseLike<any>, re: RegExp, msg?: string): PromiseLike<void>;

        property(obj: PromiseLike<Object>, prop: string, msg?: string): PromiseLike<void>;
        notProperty(obj: PromiseLike<Object>, prop: string, msg?: string): PromiseLike<void>;
        deepProperty(obj: PromiseLike<Object>, prop: string, msg?: string): PromiseLike<void>;
        notDeepProperty(obj: PromiseLike<Object>, prop: string, msg?: string): PromiseLike<void>;

        propertyVal(obj: PromiseLike<Object>, prop: string, val: any, msg?: string): PromiseLike<void>;
        propertyNotVal(obj: PromiseLike<Object>, prop: string, val: any, msg?: string): PromiseLike<void>;

        deepPropertyVal(obj: PromiseLike<Object>, prop: string, val: any, msg?: string): PromiseLike<void>;
        deepPropertyNotVal(obj: PromiseLike<Object>, prop: string, val: any, msg?: string): PromiseLike<void>;

        nestedProperty(obj: PromiseLike<object>, prop: string, msg?: string): PromiseLike<void>;
        notNestedProperty(obj: PromiseLike<object>, prop: string, msg?: string): PromiseLike<void>;
        nestedPropertyVal(obj: PromiseLike<object>, prop: string, val: any, msg?: string): PromiseLike<void>;
        notNestedPropertyVal(obj: PromiseLike<object>, prop: string, val: any, msg?: string): PromiseLike<void>;

        deepNestedPropertyVal(obj: PromiseLike<object>, prop: string, val: any, msg?: string): PromiseLike<void>;
        notDeepNestedPropertyVal(obj: PromiseLike<object>, prop: string, val: any, msg?: string): PromiseLike<void>;

        lengthOf(exp: PromiseLike<any>, len: number, msg?: string): PromiseLike<void>;

        hasAnyKeys(obj: PromiseLike<any>, keys: any[], msg?: string): PromiseLike<void>;
        hasAnyKeys(obj: PromiseLike<any>, keys: object, msg?: string): PromiseLike<void>;

        hasAllKeys(obj: PromiseLike<any>, keys: any[], msg?: string): PromiseLike<void>;
        hasAllKeys(obj: PromiseLike<any>, keys: object, msg?: string): PromiseLike<void>;

        containsAllKeys(obj: PromiseLike<any>, keys: any[], msg?: string): PromiseLike<void>;
        containsAllKeys(obj: PromiseLike<any>, keys: object, msg?: string): PromiseLike<void>;

        doesNotHaveAnyKeys(obj: PromiseLike<any>, keys: any[], msg?: string): PromiseLike<void>;
        doesNotHaveAnyKeys(obj: PromiseLike<any>, keys: object, msg?: string): PromiseLike<void>;

        doesNotHaveAllKeys(obj: PromiseLike<any>, keys: any[], msg?: string): PromiseLike<void>;
        doesNotHaveAllKeys(obj: PromiseLike<any>, keys: object, msg?: string): PromiseLike<void>;

        hasAnyDeepKeys(obj: PromiseLike<any>, keys: any[], msg?: string): PromiseLike<void>;
        hasAnyDeepKeys(obj: PromiseLike<any>, keys: object, msg?: string): PromiseLike<void>;

        hasAllDeepKeys(obj: PromiseLike<any>, keys: any[], msg?: string): PromiseLike<void>;
        hasAllDeepKeys(obj: PromiseLike<any>, keys: object, msg?: string): PromiseLike<void>;

        containsAllDeepKeys(obj: PromiseLike<any>, keys: any[], msg?: string): PromiseLike<void>;
        containsAllDeepKeys(obj: PromiseLike<any>, keys: object, msg?: string): PromiseLike<void>;

        doesNotHaveAnyDeepKeys(obj: PromiseLike<any>, keys: any[], msg?: string): PromiseLike<void>;
        doesNotHaveAnyDeepKeys(obj: PromiseLike<any>, keys: object, msg?: string): PromiseLike<void>;

        doesNotHaveAllDeepKeys(obj: PromiseLike<any>, keys: any[], msg?: string): PromiseLike<void>;
        doesNotHaveAllDeepKeys(obj: PromiseLike<any>, keys: object, msg?: string): PromiseLike<void>;

        // alias frenzy
        throw(fn: Function, msg?: string): PromiseLike<void>;
        throw(fn: Function, regExp: RegExp): PromiseLike<void>;
        throw(fn: Function, errType: Function, msg?: string): PromiseLike<void>;
        throw(fn: Function, errType: Function, regExp: RegExp): PromiseLike<void>;

        throws(fn: Function, msg?: string): PromiseLike<void>;
        throws(fn: Function, regExp: RegExp): PromiseLike<void>;
        throws(fn: Function, errType: Function, msg?: string): PromiseLike<void>;
        throws(fn: Function, errType: Function, regExp: RegExp): PromiseLike<void>;

        Throw(fn: Function, msg?: string): PromiseLike<void>;
        Throw(fn: Function, regExp: RegExp): PromiseLike<void>;
        Throw(fn: Function, errType: Function, msg?: string): PromiseLike<void>;
        Throw(fn: Function, errType: Function, regExp: RegExp): PromiseLike<void>;

        doesNotThrow(fn: Function, msg?: string): PromiseLike<void>;
        doesNotThrow(fn: Function, regExp: RegExp): PromiseLike<void>;
        doesNotThrow(fn: Function, errType: Function, msg?: string): PromiseLike<void>;
        doesNotThrow(fn: Function, errType: Function, regExp: RegExp): PromiseLike<void>;

        operator(val: PromiseLike<any>, operator: string, val2: any, msg?: string): PromiseLike<void>;
        closeTo(act: PromiseLike<number>, exp: number, delta: number, msg?: string): PromiseLike<void>;
        approximately(act: PromiseLike<number>, exp: number, delta: number, msg?: string): PromiseLike<void>;

        sameMembers(set1: PromiseLike<any[]>, set2: any[], msg?: string): PromiseLike<void>;
        sameDeepMembers(set1: PromiseLike<any[]>, set2: any[], msg?: string): PromiseLike<void>;
        sameOrderedMembers(set1: PromiseLike<any[]>, set2: any[], msg?: string): PromiseLike<void>;
        notSameOrderedMembers(set1: PromiseLike<any[]>, set2: any[], msg?: string): PromiseLike<void>;
        sameDeepOrderedMembers(set1: PromiseLike<any[]>, set2: any[], msg?: string): PromiseLike<void>;
        notSameDeepOrderedMembers(set1: PromiseLike<any[]>, set2: any[], msg?: string): PromiseLike<void>;
        includeOrderedMembers(set1: PromiseLike<any[]>, set2: any[], msg?: string): PromiseLike<void>;
        notIncludeOrderedMembers(set1: PromiseLike<any[]>, set2: any[], msg?: string): PromiseLike<void>;
        includeDeepOrderedMembers(set1: PromiseLike<any[]>, set2: any[], msg?: string): PromiseLike<void>;
        notIncludeDeepOrderedMembers(set1: PromiseLike<any[]>, set2: any[], msg?: string): PromiseLike<void>;
        includeMembers(set1: PromiseLike<any[]>, set2: any[], msg?: string): PromiseLike<void>;
        includeDeepMembers(set1: PromiseLike<any[]>, set2: any[], msg?: string): PromiseLike<void>;

        oneOf(val: PromiseLike<any>, list: any[], msg?: string): PromiseLike<void>;

        changes(modifier: Function, obj: Object, property: string, msg?: string): PromiseLike<void>;
        changesBy(modifier: Function, obj: object, property: string, change: number, msg?: string): PromiseLike<void>;
        doesNotChange(modifier: Function, obj: Object, property: string, msg?: string): PromiseLike<void>;
        changesButNotBy(
            modifier: Function,
            obj: object,
            property: string,
            change: number,
            msg?: string,
        ): PromiseLike<void>;
        increases(modifier: Function, obj: Object, property: string, msg?: string): PromiseLike<void>;
        increasesBy(modifier: Function, obj: Object, property: string, change: number, msg?: string): PromiseLike<void>;
        doesNotIncrease(modifier: Function, obj: Object, property: string, msg?: string): PromiseLike<void>;
        increasesButNotBy(
            modifier: Function,
            obj: Object,
            property: string,
            change: number,
            msg?: string,
        ): PromiseLike<void>;
        decreases(modifier: Function, obj: Object, property: string, msg?: string): PromiseLike<void>;
        decreasesBy(modifier: Function, obj: Object, property: string, change: number, msg?: string): PromiseLike<void>;
        doesNotDecrease(modifier: Function, obj: Object, property: string, msg?: string): PromiseLike<void>;
        decreasesButNotBy(
            modifier: Function,
            obj: Object,
            property: string,
            change: number,
            msg?: string,
        ): PromiseLike<void>;

        ifError(val: PromiseLike<any>, msg?: string): PromiseLike<void>;

        isExtensible(obj: PromiseLike<Object>, msg?: string): PromiseLike<void>;
        isNotExtensible(obj: PromiseLike<Object>, msg?: string): PromiseLike<void>;

        isSealed(obj: PromiseLike<Object>, msg?: string): PromiseLike<void>;
        sealed(obj: PromiseLike<Object>, msg?: string): PromiseLike<void>;
        isNotSealed(obj: PromiseLike<Object>, msg?: string): PromiseLike<void>;
        notSealed(obj: PromiseLike<Object>, msg?: string): PromiseLike<void>;

        isFrozen(obj: PromiseLike<Object>, msg?: string): PromiseLike<void>;
        frozen(obj: PromiseLike<Object>, msg?: string): PromiseLike<void>;
        isNotFrozen(obj: PromiseLike<Object>, msg?: string): PromiseLike<void>;
        notFrozen(obj: PromiseLike<Object>, msg?: string): PromiseLike<void>;

        isEmpty(val: PromiseLike<any>, msg?: string): PromiseLike<void>;
        isNotEmpty(val: PromiseLike<any>, msg?: string): PromiseLike<void>;
    }
}
