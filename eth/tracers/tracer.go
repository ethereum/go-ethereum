// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package tracers

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	duktape "gopkg.in/olebedev/go-duktape.v3"
)

// bigIntegerJS is the minified version of https://github.com/peterolson/BigInteger.js.
const bigIntegerJS = `var bigInt=function(undefined){"use strict";var BASE=1e7,LOG_BASE=7,MAX_INT=9007199254740992,MAX_INT_ARR=smallToArray(MAX_INT),LOG_MAX_INT=Math.log(MAX_INT);function Integer(v,radix){if(typeof v==="undefined")return Integer[0];if(typeof radix!=="undefined")return+radix===10?parseValue(v):parseBase(v,radix);return parseValue(v)}function BigInteger(value,sign){this.value=value;this.sign=sign;this.isSmall=false}BigInteger.prototype=Object.create(Integer.prototype);function SmallInteger(value){this.value=value;this.sign=value<0;this.isSmall=true}SmallInteger.prototype=Object.create(Integer.prototype);function isPrecise(n){return-MAX_INT<n&&n<MAX_INT}function smallToArray(n){if(n<1e7)return[n];if(n<1e14)return[n%1e7,Math.floor(n/1e7)];return[n%1e7,Math.floor(n/1e7)%1e7,Math.floor(n/1e14)]}function arrayToSmall(arr){trim(arr);var length=arr.length;if(length<4&&compareAbs(arr,MAX_INT_ARR)<0){switch(length){case 0:return 0;case 1:return arr[0];case 2:return arr[0]+arr[1]*BASE;default:return arr[0]+(arr[1]+arr[2]*BASE)*BASE}}return arr}function trim(v){var i=v.length;while(v[--i]===0);v.length=i+1}function createArray(length){var x=new Array(length);var i=-1;while(++i<length){x[i]=0}return x}function truncate(n){if(n>0)return Math.floor(n);return Math.ceil(n)}function add(a,b){var l_a=a.length,l_b=b.length,r=new Array(l_a),carry=0,base=BASE,sum,i;for(i=0;i<l_b;i++){sum=a[i]+b[i]+carry;carry=sum>=base?1:0;r[i]=sum-carry*base}while(i<l_a){sum=a[i]+carry;carry=sum===base?1:0;r[i++]=sum-carry*base}if(carry>0)r.push(carry);return r}function addAny(a,b){if(a.length>=b.length)return add(a,b);return add(b,a)}function addSmall(a,carry){var l=a.length,r=new Array(l),base=BASE,sum,i;for(i=0;i<l;i++){sum=a[i]-base+carry;carry=Math.floor(sum/base);r[i]=sum-carry*base;carry+=1}while(carry>0){r[i++]=carry%base;carry=Math.floor(carry/base)}return r}BigInteger.prototype.add=function(v){var n=parseValue(v);if(this.sign!==n.sign){return this.subtract(n.negate())}var a=this.value,b=n.value;if(n.isSmall){return new BigInteger(addSmall(a,Math.abs(b)),this.sign)}return new BigInteger(addAny(a,b),this.sign)};BigInteger.prototype.plus=BigInteger.prototype.add;SmallInteger.prototype.add=function(v){var n=parseValue(v);var a=this.value;if(a<0!==n.sign){return this.subtract(n.negate())}var b=n.value;if(n.isSmall){if(isPrecise(a+b))return new SmallInteger(a+b);b=smallToArray(Math.abs(b))}return new BigInteger(addSmall(b,Math.abs(a)),a<0)};SmallInteger.prototype.plus=SmallInteger.prototype.add;function subtract(a,b){var a_l=a.length,b_l=b.length,r=new Array(a_l),borrow=0,base=BASE,i,difference;for(i=0;i<b_l;i++){difference=a[i]-borrow-b[i];if(difference<0){difference+=base;borrow=1}else borrow=0;r[i]=difference}for(i=b_l;i<a_l;i++){difference=a[i]-borrow;if(difference<0)difference+=base;else{r[i++]=difference;break}r[i]=difference}for(;i<a_l;i++){r[i]=a[i]}trim(r);return r}function subtractAny(a,b,sign){var value;if(compareAbs(a,b)>=0){value=subtract(a,b)}else{value=subtract(b,a);sign=!sign}value=arrayToSmall(value);if(typeof value==="number"){if(sign)value=-value;return new SmallInteger(value)}return new BigInteger(value,sign)}function subtractSmall(a,b,sign){var l=a.length,r=new Array(l),carry=-b,base=BASE,i,difference;for(i=0;i<l;i++){difference=a[i]+carry;carry=Math.floor(difference/base);difference%=base;r[i]=difference<0?difference+base:difference}r=arrayToSmall(r);if(typeof r==="number"){if(sign)r=-r;return new SmallInteger(r)}return new BigInteger(r,sign)}BigInteger.prototype.subtract=function(v){var n=parseValue(v);if(this.sign!==n.sign){return this.add(n.negate())}var a=this.value,b=n.value;if(n.isSmall)return subtractSmall(a,Math.abs(b),this.sign);return subtractAny(a,b,this.sign)};BigInteger.prototype.minus=BigInteger.prototype.subtract;SmallInteger.prototype.subtract=function(v){var n=parseValue(v);var a=this.value;if(a<0!==n.sign){return this.add(n.negate())}var b=n.value;if(n.isSmall){return new SmallInteger(a-b)}return subtractSmall(b,Math.abs(a),a>=0)};SmallInteger.prototype.minus=SmallInteger.prototype.subtract;BigInteger.prototype.negate=function(){return new BigInteger(this.value,!this.sign)};SmallInteger.prototype.negate=function(){var sign=this.sign;var small=new SmallInteger(-this.value);small.sign=!sign;return small};BigInteger.prototype.abs=function(){return new BigInteger(this.value,false)};SmallInteger.prototype.abs=function(){return new SmallInteger(Math.abs(this.value))};function multiplyLong(a,b){var a_l=a.length,b_l=b.length,l=a_l+b_l,r=createArray(l),base=BASE,product,carry,i,a_i,b_j;for(i=0;i<a_l;++i){a_i=a[i];for(var j=0;j<b_l;++j){b_j=b[j];product=a_i*b_j+r[i+j];carry=Math.floor(product/base);r[i+j]=product-carry*base;r[i+j+1]+=carry}}trim(r);return r}function multiplySmall(a,b){var l=a.length,r=new Array(l),base=BASE,carry=0,product,i;for(i=0;i<l;i++){product=a[i]*b+carry;carry=Math.floor(product/base);r[i]=product-carry*base}while(carry>0){r[i++]=carry%base;carry=Math.floor(carry/base)}return r}function shiftLeft(x,n){var r=[];while(n-- >0)r.push(0);return r.concat(x)}function multiplyKaratsuba(x,y){var n=Math.max(x.length,y.length);if(n<=30)return multiplyLong(x,y);n=Math.ceil(n/2);var b=x.slice(n),a=x.slice(0,n),d=y.slice(n),c=y.slice(0,n);var ac=multiplyKaratsuba(a,c),bd=multiplyKaratsuba(b,d),abcd=multiplyKaratsuba(addAny(a,b),addAny(c,d));var product=addAny(addAny(ac,shiftLeft(subtract(subtract(abcd,ac),bd),n)),shiftLeft(bd,2*n));trim(product);return product}function useKaratsuba(l1,l2){return-.012*l1-.012*l2+15e-6*l1*l2>0}BigInteger.prototype.multiply=function(v){var n=parseValue(v),a=this.value,b=n.value,sign=this.sign!==n.sign,abs;if(n.isSmall){if(b===0)return Integer[0];if(b===1)return this;if(b===-1)return this.negate();abs=Math.abs(b);if(abs<BASE){return new BigInteger(multiplySmall(a,abs),sign)}b=smallToArray(abs)}if(useKaratsuba(a.length,b.length))return new BigInteger(multiplyKaratsuba(a,b),sign);return new BigInteger(multiplyLong(a,b),sign)};BigInteger.prototype.times=BigInteger.prototype.multiply;function multiplySmallAndArray(a,b,sign){if(a<BASE){return new BigInteger(multiplySmall(b,a),sign)}return new BigInteger(multiplyLong(b,smallToArray(a)),sign)}SmallInteger.prototype._multiplyBySmall=function(a){if(isPrecise(a.value*this.value)){return new SmallInteger(a.value*this.value)}return multiplySmallAndArray(Math.abs(a.value),smallToArray(Math.abs(this.value)),this.sign!==a.sign)};BigInteger.prototype._multiplyBySmall=function(a){if(a.value===0)return Integer[0];if(a.value===1)return this;if(a.value===-1)return this.negate();return multiplySmallAndArray(Math.abs(a.value),this.value,this.sign!==a.sign)};SmallInteger.prototype.multiply=function(v){return parseValue(v)._multiplyBySmall(this)};SmallInteger.prototype.times=SmallInteger.prototype.multiply;function square(a){var l=a.length,r=createArray(l+l),base=BASE,product,carry,i,a_i,a_j;for(i=0;i<l;i++){a_i=a[i];for(var j=0;j<l;j++){a_j=a[j];product=a_i*a_j+r[i+j];carry=Math.floor(product/base);r[i+j]=product-carry*base;r[i+j+1]+=carry}}trim(r);return r}BigInteger.prototype.square=function(){return new BigInteger(square(this.value),false)};SmallInteger.prototype.square=function(){var value=this.value*this.value;if(isPrecise(value))return new SmallInteger(value);return new BigInteger(square(smallToArray(Math.abs(this.value))),false)};function divMod1(a,b){var a_l=a.length,b_l=b.length,base=BASE,result=createArray(b.length),divisorMostSignificantDigit=b[b_l-1],lambda=Math.ceil(base/(2*divisorMostSignificantDigit)),remainder=multiplySmall(a,lambda),divisor=multiplySmall(b,lambda),quotientDigit,shift,carry,borrow,i,l,q;if(remainder.length<=a_l)remainder.push(0);divisor.push(0);divisorMostSignificantDigit=divisor[b_l-1];for(shift=a_l-b_l;shift>=0;shift--){quotientDigit=base-1;if(remainder[shift+b_l]!==divisorMostSignificantDigit){quotientDigit=Math.floor((remainder[shift+b_l]*base+remainder[shift+b_l-1])/divisorMostSignificantDigit)}carry=0;borrow=0;l=divisor.length;for(i=0;i<l;i++){carry+=quotientDigit*divisor[i];q=Math.floor(carry/base);borrow+=remainder[shift+i]-(carry-q*base);carry=q;if(borrow<0){remainder[shift+i]=borrow+base;borrow=-1}else{remainder[shift+i]=borrow;borrow=0}}while(borrow!==0){quotientDigit-=1;carry=0;for(i=0;i<l;i++){carry+=remainder[shift+i]-base+divisor[i];if(carry<0){remainder[shift+i]=carry+base;carry=0}else{remainder[shift+i]=carry;carry=1}}borrow+=carry}result[shift]=quotientDigit}remainder=divModSmall(remainder,lambda)[0];return[arrayToSmall(result),arrayToSmall(remainder)]}function divMod2(a,b){var a_l=a.length,b_l=b.length,result=[],part=[],base=BASE,guess,xlen,highx,highy,check;while(a_l){part.unshift(a[--a_l]);trim(part);if(compareAbs(part,b)<0){result.push(0);continue}xlen=part.length;highx=part[xlen-1]*base+part[xlen-2];highy=b[b_l-1]*base+b[b_l-2];if(xlen>b_l){highx=(highx+1)*base}guess=Math.ceil(highx/highy);do{check=multiplySmall(b,guess);if(compareAbs(check,part)<=0)break;guess--}while(guess);result.push(guess);part=subtract(part,check)}result.reverse();return[arrayToSmall(result),arrayToSmall(part)]}function divModSmall(value,lambda){var length=value.length,quotient=createArray(length),base=BASE,i,q,remainder,divisor;remainder=0;for(i=length-1;i>=0;--i){divisor=remainder*base+value[i];q=truncate(divisor/lambda);remainder=divisor-q*lambda;quotient[i]=q|0}return[quotient,remainder|0]}function divModAny(self,v){var value,n=parseValue(v);var a=self.value,b=n.value;var quotient;if(b===0)throw new Error("Cannot divide by zero");if(self.isSmall){if(n.isSmall){return[new SmallInteger(truncate(a/b)),new SmallInteger(a%b)]}return[Integer[0],self]}if(n.isSmall){if(b===1)return[self,Integer[0]];if(b==-1)return[self.negate(),Integer[0]];var abs=Math.abs(b);if(abs<BASE){value=divModSmall(a,abs);quotient=arrayToSmall(value[0]);var remainder=value[1];if(self.sign)remainder=-remainder;if(typeof quotient==="number"){if(self.sign!==n.sign)quotient=-quotient;return[new SmallInteger(quotient),new SmallInteger(remainder)]}return[new BigInteger(quotient,self.sign!==n.sign),new SmallInteger(remainder)]}b=smallToArray(abs)}var comparison=compareAbs(a,b);if(comparison===-1)return[Integer[0],self];if(comparison===0)return[Integer[self.sign===n.sign?1:-1],Integer[0]];if(a.length+b.length<=200)value=divMod1(a,b);else value=divMod2(a,b);quotient=value[0];var qSign=self.sign!==n.sign,mod=value[1],mSign=self.sign;if(typeof quotient==="number"){if(qSign)quotient=-quotient;quotient=new SmallInteger(quotient)}else quotient=new BigInteger(quotient,qSign);if(typeof mod==="number"){if(mSign)mod=-mod;mod=new SmallInteger(mod)}else mod=new BigInteger(mod,mSign);return[quotient,mod]}BigInteger.prototype.divmod=function(v){var result=divModAny(this,v);return{quotient:result[0],remainder:result[1]}};SmallInteger.prototype.divmod=BigInteger.prototype.divmod;BigInteger.prototype.divide=function(v){return divModAny(this,v)[0]};SmallInteger.prototype.over=SmallInteger.prototype.divide=BigInteger.prototype.over=BigInteger.prototype.divide;BigInteger.prototype.mod=function(v){return divModAny(this,v)[1]};SmallInteger.prototype.remainder=SmallInteger.prototype.mod=BigInteger.prototype.remainder=BigInteger.prototype.mod;BigInteger.prototype.pow=function(v){var n=parseValue(v),a=this.value,b=n.value,value,x,y;if(b===0)return Integer[1];if(a===0)return Integer[0];if(a===1)return Integer[1];if(a===-1)return n.isEven()?Integer[1]:Integer[-1];if(n.sign){return Integer[0]}if(!n.isSmall)throw new Error("The exponent "+n.toString()+" is too large.");if(this.isSmall){if(isPrecise(value=Math.pow(a,b)))return new SmallInteger(truncate(value))}x=this;y=Integer[1];while(true){if(b&1===1){y=y.times(x);--b}if(b===0)break;b/=2;x=x.square()}return y};SmallInteger.prototype.pow=BigInteger.prototype.pow;BigInteger.prototype.modPow=function(exp,mod){exp=parseValue(exp);mod=parseValue(mod);if(mod.isZero())throw new Error("Cannot take modPow with modulus 0");var r=Integer[1],base=this.mod(mod);while(exp.isPositive()){if(base.isZero())return Integer[0];if(exp.isOdd())r=r.multiply(base).mod(mod);exp=exp.divide(2);base=base.square().mod(mod)}return r};SmallInteger.prototype.modPow=BigInteger.prototype.modPow;function compareAbs(a,b){if(a.length!==b.length){return a.length>b.length?1:-1}for(var i=a.length-1;i>=0;i--){if(a[i]!==b[i])return a[i]>b[i]?1:-1}return 0}BigInteger.prototype.compareAbs=function(v){var n=parseValue(v),a=this.value,b=n.value;if(n.isSmall)return 1;return compareAbs(a,b)};SmallInteger.prototype.compareAbs=function(v){var n=parseValue(v),a=Math.abs(this.value),b=n.value;if(n.isSmall){b=Math.abs(b);return a===b?0:a>b?1:-1}return-1};BigInteger.prototype.compare=function(v){if(v===Infinity){return-1}if(v===-Infinity){return 1}var n=parseValue(v),a=this.value,b=n.value;if(this.sign!==n.sign){return n.sign?1:-1}if(n.isSmall){return this.sign?-1:1}return compareAbs(a,b)*(this.sign?-1:1)};BigInteger.prototype.compareTo=BigInteger.prototype.compare;SmallInteger.prototype.compare=function(v){if(v===Infinity){return-1}if(v===-Infinity){return 1}var n=parseValue(v),a=this.value,b=n.value;if(n.isSmall){return a==b?0:a>b?1:-1}if(a<0!==n.sign){return a<0?-1:1}return a<0?1:-1};SmallInteger.prototype.compareTo=SmallInteger.prototype.compare;BigInteger.prototype.equals=function(v){return this.compare(v)===0};SmallInteger.prototype.eq=SmallInteger.prototype.equals=BigInteger.prototype.eq=BigInteger.prototype.equals;BigInteger.prototype.notEquals=function(v){return this.compare(v)!==0};SmallInteger.prototype.neq=SmallInteger.prototype.notEquals=BigInteger.prototype.neq=BigInteger.prototype.notEquals;BigInteger.prototype.greater=function(v){return this.compare(v)>0};SmallInteger.prototype.gt=SmallInteger.prototype.greater=BigInteger.prototype.gt=BigInteger.prototype.greater;BigInteger.prototype.lesser=function(v){return this.compare(v)<0};SmallInteger.prototype.lt=SmallInteger.prototype.lesser=BigInteger.prototype.lt=BigInteger.prototype.lesser;BigInteger.prototype.greaterOrEquals=function(v){return this.compare(v)>=0};SmallInteger.prototype.geq=SmallInteger.prototype.greaterOrEquals=BigInteger.prototype.geq=BigInteger.prototype.greaterOrEquals;BigInteger.prototype.lesserOrEquals=function(v){return this.compare(v)<=0};SmallInteger.prototype.leq=SmallInteger.prototype.lesserOrEquals=BigInteger.prototype.leq=BigInteger.prototype.lesserOrEquals;BigInteger.prototype.isEven=function(){return(this.value[0]&1)===0};SmallInteger.prototype.isEven=function(){return(this.value&1)===0};BigInteger.prototype.isOdd=function(){return(this.value[0]&1)===1};SmallInteger.prototype.isOdd=function(){return(this.value&1)===1};BigInteger.prototype.isPositive=function(){return!this.sign};SmallInteger.prototype.isPositive=function(){return this.value>0};BigInteger.prototype.isNegative=function(){return this.sign};SmallInteger.prototype.isNegative=function(){return this.value<0};BigInteger.prototype.isUnit=function(){return false};SmallInteger.prototype.isUnit=function(){return Math.abs(this.value)===1};BigInteger.prototype.isZero=function(){return false};SmallInteger.prototype.isZero=function(){return this.value===0};BigInteger.prototype.isDivisibleBy=function(v){var n=parseValue(v);var value=n.value;if(value===0)return false;if(value===1)return true;if(value===2)return this.isEven();return this.mod(n).equals(Integer[0])};SmallInteger.prototype.isDivisibleBy=BigInteger.prototype.isDivisibleBy;function isBasicPrime(v){var n=v.abs();if(n.isUnit())return false;if(n.equals(2)||n.equals(3)||n.equals(5))return true;if(n.isEven()||n.isDivisibleBy(3)||n.isDivisibleBy(5))return false;if(n.lesser(25))return true}BigInteger.prototype.isPrime=function(){var isPrime=isBasicPrime(this);if(isPrime!==undefined)return isPrime;var n=this.abs(),nPrev=n.prev();var a=[2,3,5,7,11,13,17,19],b=nPrev,d,t,i,x;while(b.isEven())b=b.divide(2);for(i=0;i<a.length;i++){x=bigInt(a[i]).modPow(b,n);if(x.equals(Integer[1])||x.equals(nPrev))continue;for(t=true,d=b;t&&d.lesser(nPrev);d=d.multiply(2)){x=x.square().mod(n);if(x.equals(nPrev))t=false}if(t)return false}return true};SmallInteger.prototype.isPrime=BigInteger.prototype.isPrime;BigInteger.prototype.isProbablePrime=function(iterations){var isPrime=isBasicPrime(this);if(isPrime!==undefined)return isPrime;var n=this.abs();var t=iterations===undefined?5:iterations;for(var i=0;i<t;i++){var a=bigInt.randBetween(2,n.minus(2));if(!a.modPow(n.prev(),n).isUnit())return false}return true};SmallInteger.prototype.isProbablePrime=BigInteger.prototype.isProbablePrime;BigInteger.prototype.modInv=function(n){var t=bigInt.zero,newT=bigInt.one,r=parseValue(n),newR=this.abs(),q,lastT,lastR;while(!newR.equals(bigInt.zero)){q=r.divide(newR);lastT=t;lastR=r;t=newT;r=newR;newT=lastT.subtract(q.multiply(newT));newR=lastR.subtract(q.multiply(newR))}if(!r.equals(1))throw new Error(this.toString()+" and "+n.toString()+" are not co-prime");if(t.compare(0)===-1){t=t.add(n)}if(this.isNegative()){return t.negate()}return t};SmallInteger.prototype.modInv=BigInteger.prototype.modInv;BigInteger.prototype.next=function(){var value=this.value;if(this.sign){return subtractSmall(value,1,this.sign)}return new BigInteger(addSmall(value,1),this.sign)};SmallInteger.prototype.next=function(){var value=this.value;if(value+1<MAX_INT)return new SmallInteger(value+1);return new BigInteger(MAX_INT_ARR,false)};BigInteger.prototype.prev=function(){var value=this.value;if(this.sign){return new BigInteger(addSmall(value,1),true)}return subtractSmall(value,1,this.sign)};SmallInteger.prototype.prev=function(){var value=this.value;if(value-1>-MAX_INT)return new SmallInteger(value-1);return new BigInteger(MAX_INT_ARR,true)};var powersOfTwo=[1];while(2*powersOfTwo[powersOfTwo.length-1]<=BASE)powersOfTwo.push(2*powersOfTwo[powersOfTwo.length-1]);var powers2Length=powersOfTwo.length,highestPower2=powersOfTwo[powers2Length-1];function shift_isSmall(n){return(typeof n==="number"||typeof n==="string")&&+Math.abs(n)<=BASE||n instanceof BigInteger&&n.value.length<=1}BigInteger.prototype.shiftLeft=function(n){if(!shift_isSmall(n)){throw new Error(String(n)+" is too large for shifting.")}n=+n;if(n<0)return this.shiftRight(-n);var result=this;while(n>=powers2Length){result=result.multiply(highestPower2);n-=powers2Length-1}return result.multiply(powersOfTwo[n])};SmallInteger.prototype.shiftLeft=BigInteger.prototype.shiftLeft;BigInteger.prototype.shiftRight=function(n){var remQuo;if(!shift_isSmall(n)){throw new Error(String(n)+" is too large for shifting.")}n=+n;if(n<0)return this.shiftLeft(-n);var result=this;while(n>=powers2Length){if(result.isZero())return result;remQuo=divModAny(result,highestPower2);result=remQuo[1].isNegative()?remQuo[0].prev():remQuo[0];n-=powers2Length-1}remQuo=divModAny(result,powersOfTwo[n]);return remQuo[1].isNegative()?remQuo[0].prev():remQuo[0]};SmallInteger.prototype.shiftRight=BigInteger.prototype.shiftRight;function bitwise(x,y,fn){y=parseValue(y);var xSign=x.isNegative(),ySign=y.isNegative();var xRem=xSign?x.not():x,yRem=ySign?y.not():y;var xDigit=0,yDigit=0;var xDivMod=null,yDivMod=null;var result=[];while(!xRem.isZero()||!yRem.isZero()){xDivMod=divModAny(xRem,highestPower2);xDigit=xDivMod[1].toJSNumber();if(xSign){xDigit=highestPower2-1-xDigit}yDivMod=divModAny(yRem,highestPower2);yDigit=yDivMod[1].toJSNumber();if(ySign){yDigit=highestPower2-1-yDigit}xRem=xDivMod[0];yRem=yDivMod[0];result.push(fn(xDigit,yDigit))}var sum=fn(xSign?1:0,ySign?1:0)!==0?bigInt(-1):bigInt(0);for(var i=result.length-1;i>=0;i-=1){sum=sum.multiply(highestPower2).add(bigInt(result[i]))}return sum}BigInteger.prototype.not=function(){return this.negate().prev()};SmallInteger.prototype.not=BigInteger.prototype.not;BigInteger.prototype.and=function(n){return bitwise(this,n,function(a,b){return a&b})};SmallInteger.prototype.and=BigInteger.prototype.and;BigInteger.prototype.or=function(n){return bitwise(this,n,function(a,b){return a|b})};SmallInteger.prototype.or=BigInteger.prototype.or;BigInteger.prototype.xor=function(n){return bitwise(this,n,function(a,b){return a^b})};SmallInteger.prototype.xor=BigInteger.prototype.xor;var LOBMASK_I=1<<30,LOBMASK_BI=(BASE&-BASE)*(BASE&-BASE)|LOBMASK_I;function roughLOB(n){var v=n.value,x=typeof v==="number"?v|LOBMASK_I:v[0]+v[1]*BASE|LOBMASK_BI;return x&-x}function max(a,b){a=parseValue(a);b=parseValue(b);return a.greater(b)?a:b}function min(a,b){a=parseValue(a);b=parseValue(b);return a.lesser(b)?a:b}function gcd(a,b){a=parseValue(a).abs();b=parseValue(b).abs();if(a.equals(b))return a;if(a.isZero())return b;if(b.isZero())return a;var c=Integer[1],d,t;while(a.isEven()&&b.isEven()){d=Math.min(roughLOB(a),roughLOB(b));a=a.divide(d);b=b.divide(d);c=c.multiply(d)}while(a.isEven()){a=a.divide(roughLOB(a))}do{while(b.isEven()){b=b.divide(roughLOB(b))}if(a.greater(b)){t=b;b=a;a=t}b=b.subtract(a)}while(!b.isZero());return c.isUnit()?a:a.multiply(c)}function lcm(a,b){a=parseValue(a).abs();b=parseValue(b).abs();return a.divide(gcd(a,b)).multiply(b)}function randBetween(a,b){a=parseValue(a);b=parseValue(b);var low=min(a,b),high=max(a,b);var range=high.subtract(low).add(1);if(range.isSmall)return low.add(Math.floor(Math.random()*range));var length=range.value.length-1;var result=[],restricted=true;for(var i=length;i>=0;i--){var top=restricted?range.value[i]:BASE;var digit=truncate(Math.random()*top);result.unshift(digit);if(digit<top)restricted=false}result=arrayToSmall(result);return low.add(typeof result==="number"?new SmallInteger(result):new BigInteger(result,false))}var parseBase=function(text,base){var length=text.length;var i;var absBase=Math.abs(base);for(var i=0;i<length;i++){var c=text[i].toLowerCase();if(c==="-")continue;if(/[a-z0-9]/.test(c)){if(/[0-9]/.test(c)&&+c>=absBase){if(c==="1"&&absBase===1)continue;throw new Error(c+" is not a valid digit in base "+base+".")}else if(c.charCodeAt(0)-87>=absBase){throw new Error(c+" is not a valid digit in base "+base+".")}}}if(2<=base&&base<=36){if(length<=LOG_MAX_INT/Math.log(base)){var result=parseInt(text,base);if(isNaN(result)){throw new Error(c+" is not a valid digit in base "+base+".")}return new SmallInteger(parseInt(text,base))}}base=parseValue(base);var digits=[];var isNegative=text[0]==="-";for(i=isNegative?1:0;i<text.length;i++){var c=text[i].toLowerCase(),charCode=c.charCodeAt(0);if(48<=charCode&&charCode<=57)digits.push(parseValue(c));else if(97<=charCode&&charCode<=122)digits.push(parseValue(c.charCodeAt(0)-87));else if(c==="<"){var start=i;do{i++}while(text[i]!==">");digits.push(parseValue(text.slice(start+1,i)))}else throw new Error(c+" is not a valid character")}return parseBaseFromArray(digits,base,isNegative)};function parseBaseFromArray(digits,base,isNegative){var val=Integer[0],pow=Integer[1],i;for(i=digits.length-1;i>=0;i--){val=val.add(digits[i].times(pow));pow=pow.times(base)}return isNegative?val.negate():val}function stringify(digit){var v=digit.value;if(typeof v==="number")v=[v];if(v.length===1&&v[0]<=35){return"0123456789abcdefghijklmnopqrstuvwxyz".charAt(v[0])}return"<"+v+">"}function toBase(n,base){base=bigInt(base);if(base.isZero()){if(n.isZero())return"0";throw new Error("Cannot convert nonzero numbers to base 0.")}if(base.equals(-1)){if(n.isZero())return"0";if(n.isNegative())return new Array(1-n).join("10");return"1"+new Array(+n).join("01")}var minusSign="";if(n.isNegative()&&base.isPositive()){minusSign="-";n=n.abs()}if(base.equals(1)){if(n.isZero())return"0";return minusSign+new Array(+n+1).join(1)}var out=[];var left=n,divmod;while(left.isNegative()||left.compareAbs(base)>=0){divmod=left.divmod(base);left=divmod.quotient;var digit=divmod.remainder;if(digit.isNegative()){digit=base.minus(digit).abs();left=left.next()}out.push(stringify(digit))}out.push(stringify(left));return minusSign+out.reverse().join("")}BigInteger.prototype.toString=function(radix){if(radix===undefined)radix=10;if(radix!==10)return toBase(this,radix);var v=this.value,l=v.length,str=String(v[--l]),zeros="0000000",digit;while(--l>=0){digit=String(v[l]);str+=zeros.slice(digit.length)+digit}var sign=this.sign?"-":"";return sign+str};SmallInteger.prototype.toString=function(radix){if(radix===undefined)radix=10;if(radix!=10)return toBase(this,radix);return String(this.value)};BigInteger.prototype.toJSON=SmallInteger.prototype.toJSON=function(){return this.toString()};BigInteger.prototype.valueOf=function(){return+this.toString()};BigInteger.prototype.toJSNumber=BigInteger.prototype.valueOf;SmallInteger.prototype.valueOf=function(){return this.value};SmallInteger.prototype.toJSNumber=SmallInteger.prototype.valueOf;function parseStringValue(v){if(isPrecise(+v)){var x=+v;if(x===truncate(x))return new SmallInteger(x);throw"Invalid integer: "+v}var sign=v[0]==="-";if(sign)v=v.slice(1);var split=v.split(/e/i);if(split.length>2)throw new Error("Invalid integer: "+split.join("e"));if(split.length===2){var exp=split[1];if(exp[0]==="+")exp=exp.slice(1);exp=+exp;if(exp!==truncate(exp)||!isPrecise(exp))throw new Error("Invalid integer: "+exp+" is not a valid exponent.");var text=split[0];var decimalPlace=text.indexOf(".");if(decimalPlace>=0){exp-=text.length-decimalPlace-1;text=text.slice(0,decimalPlace)+text.slice(decimalPlace+1)}if(exp<0)throw new Error("Cannot include negative exponent part for integers");text+=new Array(exp+1).join("0");v=text}var isValid=/^([0-9][0-9]*)$/.test(v);if(!isValid)throw new Error("Invalid integer: "+v);var r=[],max=v.length,l=LOG_BASE,min=max-l;while(max>0){r.push(+v.slice(min,max));min-=l;if(min<0)min=0;max-=l}trim(r);return new BigInteger(r,sign)}function parseNumberValue(v){if(isPrecise(v)){if(v!==truncate(v))throw new Error(v+" is not an integer.");return new SmallInteger(v)}return parseStringValue(v.toString())}function parseValue(v){if(typeof v==="number"){return parseNumberValue(v)}if(typeof v==="string"){return parseStringValue(v)}return v}for(var i=0;i<1e3;i++){Integer[i]=new SmallInteger(i);if(i>0)Integer[-i]=new SmallInteger(-i)}Integer.one=Integer[1];Integer.zero=Integer[0];Integer.minusOne=Integer[-1];Integer.max=max;Integer.min=min;Integer.gcd=gcd;Integer.lcm=lcm;Integer.isInstance=function(x){return x instanceof BigInteger||x instanceof SmallInteger};Integer.randBetween=randBetween;Integer.fromArray=function(digits,base,isNegative){return parseBaseFromArray(digits.map(parseValue),parseValue(base||10),isNegative)};return Integer}();if(typeof module!=="undefined"&&module.hasOwnProperty("exports")){module.exports=bigInt}if(typeof define==="function"&&define.amd){define("big-integer",[],function(){return bigInt})}; bigInt`

// makeSlice convert an unsafe memory pointer with the given type into a Go byte
// slice.
//
// Note, the returned slice uses the same memory area as the input arguments.
// If those are duktape stack items, popping them off **will** make the slice
// contents change.
func makeSlice(ptr unsafe.Pointer, size uint) []byte {
	var sl = struct {
		addr uintptr
		len  int
		cap  int
	}{uintptr(ptr), int(size), int(size)}

	return *(*[]byte)(unsafe.Pointer(&sl))
}

// popSlice pops a buffer off the JavaScript stack and returns it as a slice.
func popSlice(ctx *duktape.Context) []byte {
	blob := common.CopyBytes(makeSlice(ctx.GetBuffer(-1)))
	ctx.Pop()
	return blob
}

// pushBigInt create a JavaScript BigInteger in the VM.
func pushBigInt(n *big.Int, ctx *duktape.Context) {
	ctx.GetGlobalString("bigInt")
	ctx.PushString(n.String())
	ctx.Call(1)
}

// opWrapper provides a JavaScript wrapper around OpCode.
type opWrapper struct {
	op vm.OpCode
}

// pushObject assembles a JSVM object wrapping a swappable opcode and pushes it
// onto the VM stack.
func (ow *opWrapper) pushObject(vm *duktape.Context) {
	obj := vm.PushObject()

	vm.PushGoFunction(func(ctx *duktape.Context) int { ctx.PushInt(int(ow.op)); return 1 })
	vm.PutPropString(obj, "toNumber")

	vm.PushGoFunction(func(ctx *duktape.Context) int { ctx.PushString(ow.op.String()); return 1 })
	vm.PutPropString(obj, "toString")

	vm.PushGoFunction(func(ctx *duktape.Context) int { ctx.PushBoolean(ow.op.IsPush()); return 1 })
	vm.PutPropString(obj, "isPush")
}

// memoryWrapper provides a JavaScript wrapper around vm.Memory.
type memoryWrapper struct {
	memory *vm.Memory
}

// slice returns the requested range of memory as a byte slice.
func (mw *memoryWrapper) slice(begin, end int64) []byte {
	if mw.memory.Len() < int(end) {
		// TODO(karalabe): We can't js-throw from Go inside duktape inside Go. The Go
		// runtime goes belly up https://github.com/golang/go/issues/15639.
		log.Warn("Tracer accessed out of bound memory", "available", mw.memory.Len(), "offset", begin, "size", end-begin)
		return nil
	}
	return mw.memory.Get(begin, end-begin)
}

// getUint returns the 32 bytes at the specified address interpreted as a uint.
func (mw *memoryWrapper) getUint(addr int64) *big.Int {
	if mw.memory.Len() < int(addr)+32 {
		// TODO(karalabe): We can't js-throw from Go inside duktape inside Go. The Go
		// runtime goes belly up https://github.com/golang/go/issues/15639.
		log.Warn("Tracer accessed out of bound memory", "available", mw.memory.Len(), "offset", addr, "size", 32)
		return new(big.Int)
	}
	return new(big.Int).SetBytes(mw.memory.GetPtr(addr, 32))
}

// pushObject assembles a JSVM object wrapping a swappable memory and pushes it
// onto the VM stack.
func (mw *memoryWrapper) pushObject(vm *duktape.Context) {
	obj := vm.PushObject()

	// Generate the `slice` method which takes two ints and returns a buffer
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		blob := mw.slice(int64(ctx.GetInt(-2)), int64(ctx.GetInt(-1)))
		ctx.Pop2()

		ptr := ctx.PushFixedBuffer(len(blob))
		copy(makeSlice(ptr, uint(len(blob))), blob[:])
		return 1
	})
	vm.PutPropString(obj, "slice")

	// Generate the `getUint` method which takes an int and returns a bigint
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		offset := int64(ctx.GetInt(-1))
		ctx.Pop()

		pushBigInt(mw.getUint(offset), ctx)
		return 1
	})
	vm.PutPropString(obj, "getUint")
}

// stackWrapper provides a JavaScript wrapper around vm.Stack.
type stackWrapper struct {
	stack *vm.Stack
}

// peek returns the nth-from-the-top element of the stack.
func (sw *stackWrapper) peek(idx int) *big.Int {
	if len(sw.stack.Data()) <= idx {
		// TODO(karalabe): We can't js-throw from Go inside duktape inside Go. The Go
		// runtime goes belly up https://github.com/golang/go/issues/15639.
		log.Warn("Tracer accessed out of bound stack", "size", len(sw.stack.Data()), "index", idx)
		return new(big.Int)
	}
	return sw.stack.Data()[len(sw.stack.Data())-idx-1]
}

// pushObject assembles a JSVM object wrapping a swappable stack and pushes it
// onto the VM stack.
func (sw *stackWrapper) pushObject(vm *duktape.Context) {
	obj := vm.PushObject()

	vm.PushGoFunction(func(ctx *duktape.Context) int { ctx.PushInt(len(sw.stack.Data())); return 1 })
	vm.PutPropString(obj, "length")

	// Generate the `peek` method which takes an int and returns a bigint
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		offset := ctx.GetInt(-1)
		ctx.Pop()

		pushBigInt(sw.peek(offset), ctx)
		return 1
	})
	vm.PutPropString(obj, "peek")
}

// dbWrapper provides a JavaScript wrapper around vm.Database.
type dbWrapper struct {
	db vm.StateDB
}

// pushObject assembles a JSVM object wrapping a swappable database and pushes it
// onto the VM stack.
func (dw *dbWrapper) pushObject(vm *duktape.Context) {
	obj := vm.PushObject()

	// Push the wrapper for statedb.GetBalance
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		pushBigInt(dw.db.GetBalance(common.BytesToAddress(popSlice(ctx))), ctx)
		return 1
	})
	vm.PutPropString(obj, "getBalance")

	// Push the wrapper for statedb.GetNonce
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		ctx.PushInt(int(dw.db.GetNonce(common.BytesToAddress(popSlice(ctx)))))
		return 1
	})
	vm.PutPropString(obj, "getNonce")

	// Push the wrapper for statedb.GetCode
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		code := dw.db.GetCode(common.BytesToAddress(popSlice(ctx)))

		ptr := ctx.PushFixedBuffer(len(code))
		copy(makeSlice(ptr, uint(len(code))), code[:])
		return 1
	})
	vm.PutPropString(obj, "getCode")

	// Push the wrapper for statedb.GetState
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		hash := popSlice(ctx)
		addr := popSlice(ctx)

		state := dw.db.GetState(common.BytesToAddress(addr), common.BytesToHash(hash))

		ptr := ctx.PushFixedBuffer(len(state))
		copy(makeSlice(ptr, uint(len(state))), state[:])
		return 1
	})
	vm.PutPropString(obj, "getState")

	// Push the wrapper for statedb.Exists
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		ctx.PushBoolean(dw.db.Exist(common.BytesToAddress(popSlice(ctx))))
		return 1
	})
	vm.PutPropString(obj, "exists")
}

// contractWrapper provides a JavaScript wrapper around vm.Contract
type contractWrapper struct {
	contract *vm.Contract
}

// pushObject assembles a JSVM object wrapping a swappable contract and pushes it
// onto the VM stack.
func (cw *contractWrapper) pushObject(vm *duktape.Context) {
	obj := vm.PushObject()

	// Push the wrapper for contract.Caller
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		ptr := ctx.PushFixedBuffer(20)
		copy(makeSlice(ptr, 20), cw.contract.Caller().Bytes())
		return 1
	})
	vm.PutPropString(obj, "getCaller")

	// Push the wrapper for contract.Address
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		ptr := ctx.PushFixedBuffer(20)
		copy(makeSlice(ptr, 20), cw.contract.Address().Bytes())
		return 1
	})
	vm.PutPropString(obj, "getAddress")

	// Push the wrapper for contract.Value
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		pushBigInt(cw.contract.Value(), ctx)
		return 1
	})
	vm.PutPropString(obj, "getValue")

	// Push the wrapper for contract.Input
	vm.PushGoFunction(func(ctx *duktape.Context) int {
		blob := cw.contract.Input

		ptr := ctx.PushFixedBuffer(len(blob))
		copy(makeSlice(ptr, uint(len(blob))), blob[:])
		return 1
	})
	vm.PutPropString(obj, "getInput")
}

// Tracer provides an implementation of Tracer that evaluates a Javascript
// function for each VM execution step.
type Tracer struct {
	inited bool // Flag whether the context was already inited from the EVM

	vm *duktape.Context // Javascript VM instance

	tracerObject int // Stack index of the tracer JavaScript object
	stateObject  int // Stack index of the global state to pull arguments from

	opWrapper       *opWrapper       // Wrapper around the VM opcode
	stackWrapper    *stackWrapper    // Wrapper around the VM stack
	memoryWrapper   *memoryWrapper   // Wrapper around the VM memory
	contractWrapper *contractWrapper // Wrapper around the contract object
	dbWrapper       *dbWrapper       // Wrapper around the VM environment

	pcValue    *uint   // Swappable pc value wrapped by a log accessor
	gasValue   *uint   // Swappable gas value wrapped by a log accessor
	costValue  *uint   // Swappable cost value wrapped by a log accessor
	depthValue *uint   // Swappable depth value wrapped by a log accessor
	errorValue *string // Swappable error value wrapped by a log accessor

	ctx map[string]interface{} // Transaction context gathered throughout execution
	err error                  // Error, if one has occurred

	interrupt uint32 // Atomic flag to signal execution interruption
	reason    error  // Textual reason for the interruption
}

// New instantiates a new tracer instance. code specifies a Javascript snippet,
// which must evaluate to an expression returning an object with 'step', 'fault'
// and 'result' functions.
func New(code string) (*Tracer, error) {
	// Resolve any tracers by name and assemble the tracer object
	if tracer, ok := tracer(code); ok {
		code = tracer
	}
	tracer := &Tracer{
		vm:              duktape.New(),
		ctx:             make(map[string]interface{}),
		opWrapper:       new(opWrapper),
		stackWrapper:    new(stackWrapper),
		memoryWrapper:   new(memoryWrapper),
		contractWrapper: new(contractWrapper),
		dbWrapper:       new(dbWrapper),
		pcValue:         new(uint),
		gasValue:        new(uint),
		costValue:       new(uint),
		depthValue:      new(uint),
	}
	// Set up builtins for this environment
	tracer.vm.PushGlobalGoFunction("toHex", func(ctx *duktape.Context) int {
		ctx.PushString(hexutil.Encode(popSlice(ctx)))
		return 1
	})
	tracer.vm.PushGlobalGoFunction("toWord", func(ctx *duktape.Context) int {
		var word common.Hash
		if ptr, size := ctx.GetBuffer(-1); ptr != nil {
			word = common.BytesToHash(makeSlice(ptr, size))
		} else {
			word = common.HexToHash(ctx.GetString(-1))
		}
		ctx.Pop()
		copy(makeSlice(ctx.PushFixedBuffer(32), 32), word[:])
		return 1
	})
	tracer.vm.PushGlobalGoFunction("toAddress", func(ctx *duktape.Context) int {
		var addr common.Address
		if ptr, size := ctx.GetBuffer(-1); ptr != nil {
			addr = common.BytesToAddress(makeSlice(ptr, size))
		} else {
			addr = common.HexToAddress(ctx.GetString(-1))
		}
		ctx.Pop()
		copy(makeSlice(ctx.PushFixedBuffer(20), 20), addr[:])
		return 1
	})
	tracer.vm.PushGlobalGoFunction("toContract", func(ctx *duktape.Context) int {
		var from common.Address
		if ptr, size := ctx.GetBuffer(-2); ptr != nil {
			from = common.BytesToAddress(makeSlice(ptr, size))
		} else {
			from = common.HexToAddress(ctx.GetString(-2))
		}
		nonce := uint64(ctx.GetInt(-1))
		ctx.Pop2()

		contract := crypto.CreateAddress(from, nonce)
		copy(makeSlice(ctx.PushFixedBuffer(20), 20), contract[:])
		return 1
	})
	tracer.vm.PushGlobalGoFunction("isPrecompiled", func(ctx *duktape.Context) int {
		_, ok := vm.PrecompiledContractsByzantium[common.BytesToAddress(popSlice(ctx))]
		ctx.PushBoolean(ok)
		return 1
	})
	tracer.vm.PushGlobalGoFunction("slice", func(ctx *duktape.Context) int {
		start, end := ctx.GetInt(-2), ctx.GetInt(-1)
		ctx.Pop2()

		blob := popSlice(ctx)
		size := end - start

		if start < 0 || start > end || end > len(blob) {
			// TODO(karalabe): We can't js-throw from Go inside duktape inside Go. The Go
			// runtime goes belly up https://github.com/golang/go/issues/15639.
			log.Warn("Tracer accessed out of bound memory", "available", len(blob), "offset", start, "size", size)
			ctx.PushFixedBuffer(0)
			return 1
		}
		copy(makeSlice(ctx.PushFixedBuffer(size), uint(size)), blob[start:end])
		return 1
	})
	// Push the JavaScript tracer as object #0 onto the JSVM stack and validate it
	if err := tracer.vm.PevalString("(" + code + ")"); err != nil {
		log.Warn("Failed to compile tracer", "err", err)
		return nil, err
	}
	tracer.tracerObject = 0 // yeah, nice, eval can't return the index itself

	if !tracer.vm.GetPropString(tracer.tracerObject, "step") {
		return nil, fmt.Errorf("Trace object must expose a function step()")
	}
	tracer.vm.Pop()

	if !tracer.vm.GetPropString(tracer.tracerObject, "fault") {
		return nil, fmt.Errorf("Trace object must expose a function fault()")
	}
	tracer.vm.Pop()

	if !tracer.vm.GetPropString(tracer.tracerObject, "result") {
		return nil, fmt.Errorf("Trace object must expose a function result()")
	}
	tracer.vm.Pop()

	// Tracer is valid, inject the big int library to access large numbers
	tracer.vm.EvalString(bigIntegerJS)
	tracer.vm.PutGlobalString("bigInt")

	// Push the global environment state as object #1 into the JSVM stack
	tracer.stateObject = tracer.vm.PushObject()

	logObject := tracer.vm.PushObject()

	tracer.opWrapper.pushObject(tracer.vm)
	tracer.vm.PutPropString(logObject, "op")

	tracer.stackWrapper.pushObject(tracer.vm)
	tracer.vm.PutPropString(logObject, "stack")

	tracer.memoryWrapper.pushObject(tracer.vm)
	tracer.vm.PutPropString(logObject, "memory")

	tracer.contractWrapper.pushObject(tracer.vm)
	tracer.vm.PutPropString(logObject, "contract")

	tracer.vm.PushGoFunction(func(ctx *duktape.Context) int { ctx.PushUint(*tracer.pcValue); return 1 })
	tracer.vm.PutPropString(logObject, "getPC")

	tracer.vm.PushGoFunction(func(ctx *duktape.Context) int { ctx.PushUint(*tracer.gasValue); return 1 })
	tracer.vm.PutPropString(logObject, "getGas")

	tracer.vm.PushGoFunction(func(ctx *duktape.Context) int { ctx.PushUint(*tracer.costValue); return 1 })
	tracer.vm.PutPropString(logObject, "getCost")

	tracer.vm.PushGoFunction(func(ctx *duktape.Context) int { ctx.PushUint(*tracer.depthValue); return 1 })
	tracer.vm.PutPropString(logObject, "getDepth")

	tracer.vm.PushGoFunction(func(ctx *duktape.Context) int {
		if tracer.errorValue != nil {
			ctx.PushString(*tracer.errorValue)
		} else {
			ctx.PushUndefined()
		}
		return 1
	})
	tracer.vm.PutPropString(logObject, "getError")

	tracer.vm.PutPropString(tracer.stateObject, "log")

	tracer.dbWrapper.pushObject(tracer.vm)
	tracer.vm.PutPropString(tracer.stateObject, "db")

	return tracer, nil
}

// Stop terminates execution of the tracer at the first opportune moment.
func (jst *Tracer) Stop(err error) {
	jst.reason = err
	atomic.StoreUint32(&jst.interrupt, 1)
}

// call executes a method on a JS object, catching any errors, formatting and
// returning them as error objects.
func (jst *Tracer) call(method string, args ...string) (json.RawMessage, error) {
	// Execute the JavaScript call and return any error
	jst.vm.PushString(method)
	for _, arg := range args {
		jst.vm.GetPropString(jst.stateObject, arg)
	}
	code := jst.vm.PcallProp(jst.tracerObject, len(args))
	defer jst.vm.Pop()

	if code != 0 {
		err := jst.vm.SafeToString(-1)
		return nil, errors.New(err)
	}
	// No error occurred, extract return value and return
	return json.RawMessage(jst.vm.JsonEncode(-1)), nil
}

func wrapError(context string, err error) error {
	return fmt.Errorf("%v    in server-side tracer function '%v'", err, context)
}

// CaptureStart implements the Tracer interface to initialize the tracing operation.
func (jst *Tracer) CaptureStart(from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) error {
	jst.ctx["type"] = "CALL"
	if create {
		jst.ctx["type"] = "CREATE"
	}
	jst.ctx["from"] = from
	jst.ctx["to"] = to
	jst.ctx["input"] = input
	jst.ctx["gas"] = gas
	jst.ctx["value"] = value

	return nil
}

// CaptureState implements the Tracer interface to trace a single step of VM execution.
func (jst *Tracer) CaptureState(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, memory *vm.Memory, stack *vm.Stack, contract *vm.Contract, depth int, err error) error {
	if jst.err == nil {
		// Initialize the context if it wasn't done yet
		if !jst.inited {
			jst.ctx["block"] = env.BlockNumber.Uint64()
			jst.inited = true
		}
		// If tracing was interrupted, set the error and stop
		if atomic.LoadUint32(&jst.interrupt) > 0 {
			jst.err = jst.reason
			return nil
		}
		jst.opWrapper.op = op
		jst.stackWrapper.stack = stack
		jst.memoryWrapper.memory = memory
		jst.contractWrapper.contract = contract
		jst.dbWrapper.db = env.StateDB

		*jst.pcValue = uint(pc)
		*jst.gasValue = uint(gas)
		*jst.costValue = uint(cost)
		*jst.depthValue = uint(depth)

		jst.errorValue = nil
		if err != nil {
			jst.errorValue = new(string)
			*jst.errorValue = err.Error()
		}
		_, err := jst.call("step", "log", "db")
		if err != nil {
			jst.err = wrapError("step", err)
		}
	}
	return nil
}

// CaptureFault implements the Tracer interface to trace an execution fault
// while running an opcode.
func (jst *Tracer) CaptureFault(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, memory *vm.Memory, stack *vm.Stack, contract *vm.Contract, depth int, err error) error {
	if jst.err == nil {
		// Apart from the error, everything matches the previous invocation
		jst.errorValue = new(string)
		*jst.errorValue = err.Error()

		_, err := jst.call("fault", "log", "db")
		if err != nil {
			jst.err = wrapError("fault", err)
		}
	}
	return nil
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (jst *Tracer) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) error {
	jst.ctx["output"] = output
	jst.ctx["gasUsed"] = gasUsed
	jst.ctx["time"] = t.String()

	if err != nil {
		jst.ctx["error"] = err.Error()
	}
	return nil
}

// GetResult calls the Javascript 'result' function and returns its value, or any accumulated error
func (jst *Tracer) GetResult() (json.RawMessage, error) {
	// Transform the context into a JavaScript object and inject into the state
	obj := jst.vm.PushObject()

	for key, val := range jst.ctx {
		switch val := val.(type) {
		case uint64:
			jst.vm.PushUint(uint(val))

		case string:
			jst.vm.PushString(val)

		case []byte:
			ptr := jst.vm.PushFixedBuffer(len(val))
			copy(makeSlice(ptr, uint(len(val))), val[:])

		case common.Address:
			ptr := jst.vm.PushFixedBuffer(20)
			copy(makeSlice(ptr, 20), val[:])

		case *big.Int:
			pushBigInt(val, jst.vm)

		default:
			panic(fmt.Sprintf("unsupported type: %T", val))
		}
		jst.vm.PutPropString(obj, key)
	}
	jst.vm.PutPropString(jst.stateObject, "ctx")

	// Finalize the trace and return the results
	result, err := jst.call("result", "ctx", "db")
	if err != nil {
		jst.err = wrapError("result", err)
	}
	// Clean up the JavaScript environment
	jst.vm.DestroyHeap()
	jst.vm.Destroy()

	return result, jst.err
}
