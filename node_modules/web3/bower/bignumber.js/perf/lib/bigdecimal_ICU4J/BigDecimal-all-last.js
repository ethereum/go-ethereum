/** @license Copyright (c) 2012 Daniel Trebbien and other contributors
Portions Copyright (c) 2003 STZ-IDA and PTV AG, Karlsruhe, Germany
Portions Copyright (c) 1995-2001 International Business Machines Corporation and others

All rights reserved.

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, provided that the above copyright notice(s) and this permission notice appear in all copies of the Software and that both the above copyright notice(s) and this permission notice appear in supporting documentation.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT OF THIRD PARTY RIGHTS. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR HOLDERS INCLUDED IN THIS NOTICE BE LIABLE FOR ANY CLAIM, OR ANY SPECIAL INDIRECT OR CONSEQUENTIAL DAMAGES, OR ANY DAMAGES WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

Except as contained in this notice, the name of a copyright holder shall not be used in advertising or otherwise to promote the sale, use or other dealings in this Software without prior written authorization of the copyright holder.
*/
(function () {

var MathContext = (function () {
/* Generated from 'MathContext.nrx' 8 Sep 2000 11:07:48 [v2.00] */
/* Options: Binary Comments Crossref Format Java Logo Strictargs Strictcase Trace2 Verbose3 */
//--package com.ibm.icu.math;

/* ------------------------------------------------------------------ */
/* MathContext -- Math context settings                               */
/* ------------------------------------------------------------------ */
/* Copyright IBM Corporation, 1997, 2000.  All Rights Reserved.       */
/*                                                                    */
/*   The MathContext object encapsulates the settings used by the     */
/*   BigDecimal class; it could also be used by other arithmetics.    */
/* ------------------------------------------------------------------ */
/* Notes:                                                             */
/*                                                                    */
/* 1. The properties are checked for validity on construction, so     */
/*    the BigDecimal class may assume that they are correct.          */
/* ------------------------------------------------------------------ */
/* Author:    Mike Cowlishaw                                          */
/* 1997.09.03 Initial version (edited from netrexx.lang.RexxSet)      */
/* 1997.09.12 Add lostDigits property                                 */
/* 1998.05.02 Make the class immutable and final; drop set methods    */
/* 1998.06.05 Add Round (rounding modes) property                     */
/* 1998.06.25 Rename from DecimalContext; allow digits=0              */
/* 1998.10.12 change to com.ibm.icu.math package                          */
/* 1999.02.06 add javadoc comments                                    */
/* 1999.03.05 simplify; changes from discussion with J. Bloch         */
/* 1999.03.13 1.00 release to IBM Centre for Java Technology          */
/* 1999.07.10 1.04 flag serialization unused                          */
/* 2000.01.01 1.06 copyright update                                   */
/* ------------------------------------------------------------------ */


/* JavaScript conversion (c) 2003 STZ-IDA and PTV AG, Karlsruhe, Germany */



/**
 * The <code>MathContext</code> immutable class encapsulates the
 * settings understood by the operator methods of the {@link BigDecimal}
 * class (and potentially other classes).  Operator methods are those
 * that effect an operation on a number or a pair of numbers.
 * <p>
 * The settings, which are not base-dependent, comprise:
 * <ol>
 * <li><code>digits</code>:
 * the number of digits (precision) to be used for an operation
 * <li><code>form</code>:
 * the form of any exponent that results from the operation
 * <li><code>lostDigits</code>:
 * whether checking for lost digits is enabled
 * <li><code>roundingMode</code>:
 * the algorithm to be used for rounding.
 * </ol>
 * <p>
 * When provided, a <code>MathContext</code> object supplies the
 * settings for an operation directly.
 * <p>
 * When <code>MathContext.DEFAULT</code> is provided for a
 * <code>MathContext</code> parameter then the default settings are used
 * (<code>9, SCIENTIFIC, false, ROUND_HALF_UP</code>).
 * <p>
 * In the <code>BigDecimal</code> class, all methods which accept a
 * <code>MathContext</code> object defaults) also have a version of the
 * method which does not accept a MathContext parameter.  These versions
 * carry out unlimited precision fixed point arithmetic (as though the
 * settings were (<code>0, PLAIN, false, ROUND_HALF_UP</code>).
 * <p>
 * The instance variables are shared with default access (so they are
 * directly accessible to the <code>BigDecimal</code> class), but must
 * never be changed.
 * <p>
 * The rounding mode constants have the same names and values as the
 * constants of the same name in <code>java.math.BigDecimal</code>, to
 * maintain compatibility with earlier versions of
 * <code>BigDecimal</code>.
 *
 * @see     BigDecimal
 * @author  Mike Cowlishaw
 * @stable ICU 2.0
 */

//--public final class MathContext implements java.io.Serializable{
 //--private static final java.lang.String $0="MathContext.nrx";

 //-- methods
 MathContext.prototype.getDigits = getDigits;
 MathContext.prototype.getForm = getForm;
 MathContext.prototype.getLostDigits = getLostDigits;
 MathContext.prototype.getRoundingMode = getRoundingMode;
 MathContext.prototype.toString = toString;
 MathContext.prototype.isValidRound = isValidRound;


 /* ----- Properties ----- */
 /* properties public constant */
 /**
  * Plain (fixed point) notation, without any exponent.
  * Used as a setting to control the form of the result of a
  * <code>BigDecimal</code> operation.
  * A zero result in plain form may have a decimal part of one or
  * more zeros.
  *
  * @see #ENGINEERING
  * @see #SCIENTIFIC
  * @stable ICU 2.0
  */
 //--public static final int PLAIN=0; // [no exponent]
 MathContext.PLAIN = MathContext.prototype.PLAIN = 0; // [no exponent]

 /**
  * Standard floating point notation (with scientific exponential
  * format, where there is one digit before any decimal point).
  * Used as a setting to control the form of the result of a
  * <code>BigDecimal</code> operation.
  * A zero result in plain form may have a decimal part of one or
  * more zeros.
  *
  * @see #ENGINEERING
  * @see #PLAIN
  * @stable ICU 2.0
  */
 //--public static final int SCIENTIFIC=1; // 1 digit before .
 MathContext.SCIENTIFIC = MathContext.prototype.SCIENTIFIC = 1; // 1 digit before .

 /**
  * Standard floating point notation (with engineering exponential
  * format, where the power of ten is a multiple of 3).
  * Used as a setting to control the form of the result of a
  * <code>BigDecimal</code> operation.
  * A zero result in plain form may have a decimal part of one or
  * more zeros.
  *
  * @see #PLAIN
  * @see #SCIENTIFIC
  * @stable ICU 2.0
  */
 //--public static final int ENGINEERING=2; // 1-3 digits before .
 MathContext.ENGINEERING = MathContext.prototype.ENGINEERING = 2; // 1-3 digits before .

 // The rounding modes match the original BigDecimal class values
 /**
  * Rounding mode to round to a more positive number.
  * Used as a setting to control the rounding mode used during a
  * <code>BigDecimal</code> operation.
  * <p>
  * If any of the discarded digits are non-zero then the result
  * should be rounded towards the next more positive digit.
  * @stable ICU 2.0
  */
 //--public static final int ROUND_CEILING=2;
 MathContext.ROUND_CEILING = MathContext.prototype.ROUND_CEILING = 2;

 /**
  * Rounding mode to round towards zero.
  * Used as a setting to control the rounding mode used during a
  * <code>BigDecimal</code> operation.
  * <p>
  * All discarded digits are ignored (truncated).  The result is
  * neither incremented nor decremented.
  * @stable ICU 2.0
  */
 //--public static final int ROUND_DOWN=1;
 MathContext.ROUND_DOWN = MathContext.prototype.ROUND_DOWN = 1;

 /**
  * Rounding mode to round to a more negative number.
  * Used as a setting to control the rounding mode used during a
  * <code>BigDecimal</code> operation.
  * <p>
  * If any of the discarded digits are non-zero then the result
  * should be rounded towards the next more negative digit.
  * @stable ICU 2.0
  */
 //--public static final int ROUND_FLOOR=3;
 MathContext.ROUND_FLOOR = MathContext.prototype.ROUND_FLOOR = 3;

 /**
  * Rounding mode to round to nearest neighbor, where an equidistant
  * value is rounded down.
  * Used as a setting to control the rounding mode used during a
  * <code>BigDecimal</code> operation.
  * <p>
  * If the discarded digits represent greater than half (0.5 times)
  * the value of a one in the next position then the result should be
  * rounded up (away from zero).  Otherwise the discarded digits are
  * ignored.
  * @stable ICU 2.0
  */
 //--public static final int ROUND_HALF_DOWN=5;
 MathContext.ROUND_HALF_DOWN = MathContext.prototype.ROUND_HALF_DOWN = 5;

 /**
  * Rounding mode to round to nearest neighbor, where an equidistant
  * value is rounded to the nearest even neighbor.
  * Used as a setting to control the rounding mode used during a
  * <code>BigDecimal</code> operation.
  * <p>
  * If the discarded digits represent greater than half (0.5 times)
  * the value of a one in the next position then the result should be
  * rounded up (away from zero).  If they represent less than half,
  * then the result should be rounded down.
  * <p>
  * Otherwise (they represent exactly half) the result is rounded
  * down if its rightmost digit is even, or rounded up if its
  * rightmost digit is odd (to make an even digit).
  * @stable ICU 2.0
  */
 //--public static final int ROUND_HALF_EVEN=6;
 MathContext.ROUND_HALF_EVEN = MathContext.prototype.ROUND_HALF_EVEN = 6;

 /**
  * Rounding mode to round to nearest neighbor, where an equidistant
  * value is rounded up.
  * Used as a setting to control the rounding mode used during a
  * <code>BigDecimal</code> operation.
  * <p>
  * If the discarded digits represent greater than or equal to half
  * (0.5 times) the value of a one in the next position then the result
  * should be rounded up (away from zero).  Otherwise the discarded
  * digits are ignored.
  * @stable ICU 2.0
  */
 //--public static final int ROUND_HALF_UP=4;
 MathContext.ROUND_HALF_UP = MathContext.prototype.ROUND_HALF_UP = 4;

 /**
  * Rounding mode to assert that no rounding is necessary.
  * Used as a setting to control the rounding mode used during a
  * <code>BigDecimal</code> operation.
  * <p>
  * Rounding (potential loss of information) is not permitted.
  * If any of the discarded digits are non-zero then an
  * <code>ArithmeticException</code> should be thrown.
  * @stable ICU 2.0
  */
 //--public static final int ROUND_UNNECESSARY=7;
 MathContext.ROUND_UNNECESSARY = MathContext.prototype.ROUND_UNNECESSARY = 7;

 /**
  * Rounding mode to round away from zero.
  * Used as a setting to control the rounding mode used during a
  * <code>BigDecimal</code> operation.
  * <p>
  * If any of the discarded digits are non-zero then the result will
  * be rounded up (away from zero).
  * @stable ICU 2.0
  */
 //--public static final int ROUND_UP=0;
 MathContext.ROUND_UP = MathContext.prototype.ROUND_UP = 0;


 /* properties shared */
 /**
  * The number of digits (precision) to be used for an operation.
  * A value of 0 indicates that unlimited precision (as many digits
  * as are required) will be used.
  * <p>
  * The {@link BigDecimal} operator methods use this value to
  * determine the precision of results.
  * Note that leading zeros (in the integer part of a number) are
  * never significant.
  * <p>
  * <code>digits</code> will always be non-negative.
  *
  * @serial
  */
 //--int digits;

 /**
  * The form of results from an operation.
  * <p>
  * The {@link BigDecimal} operator methods use this value to
  * determine the form of results, in particular whether and how
  * exponential notation should be used.
  *
  * @see #ENGINEERING
  * @see #PLAIN
  * @see #SCIENTIFIC
  * @serial
  */
 //--int form; // values for this must fit in a byte

 /**
  * Controls whether lost digits checking is enabled for an
  * operation.
  * Set to <code>true</code> to enable checking, or
  * to <code>false</code> to disable checking.
  * <p>
  * When enabled, the {@link BigDecimal} operator methods check
  * the precision of their operand or operands, and throw an
  * <code>ArithmeticException</code> if an operand is more precise
  * than the digits setting (that is, digits would be lost).
  * When disabled, operands are rounded to the specified digits.
  *
  * @serial
  */
 //--boolean lostDigits;

 /**
  * The rounding algorithm to be used for an operation.
  * <p>
  * The {@link BigDecimal} operator methods use this value to
  * determine the algorithm to be used when non-zero digits have to
  * be discarded in order to reduce the precision of a result.
  * The value must be one of the public constants whose name starts
  * with <code>ROUND_</code>.
  *
  * @see #ROUND_CEILING
  * @see #ROUND_DOWN
  * @see #ROUND_FLOOR
  * @see #ROUND_HALF_DOWN
  * @see #ROUND_HALF_EVEN
  * @see #ROUND_HALF_UP
  * @see #ROUND_UNNECESSARY
  * @see #ROUND_UP
  * @serial
  */
 //--int roundingMode;

 /* properties private constant */
 // default settings
 //--private static final int DEFAULT_FORM=SCIENTIFIC;
 //--private static final int DEFAULT_DIGITS=9;
 //--private static final boolean DEFAULT_LOSTDIGITS=false;
 //--private static final int DEFAULT_ROUNDINGMODE=ROUND_HALF_UP;
 MathContext.prototype.DEFAULT_FORM=MathContext.prototype.SCIENTIFIC;
 MathContext.prototype.DEFAULT_DIGITS=9;
 MathContext.prototype.DEFAULT_LOSTDIGITS=false;
 MathContext.prototype.DEFAULT_ROUNDINGMODE=MathContext.prototype.ROUND_HALF_UP;

 /* properties private constant */

 //--private static final int MIN_DIGITS=0; // smallest value for DIGITS.
 //--private static final int MAX_DIGITS=999999999; // largest value for DIGITS.  If increased,
 MathContext.prototype.MIN_DIGITS=0; // smallest value for DIGITS.
 MathContext.prototype.MAX_DIGITS=999999999; // largest value for DIGITS.  If increased,
 // the BigDecimal class may need update.
 // list of valid rounding mode values, most common two first
 //--private static final int ROUNDS[]=new int[]{ROUND_HALF_UP,ROUND_UNNECESSARY,ROUND_CEILING,ROUND_DOWN,ROUND_FLOOR,ROUND_HALF_DOWN,ROUND_HALF_EVEN,ROUND_UP};
 MathContext.prototype.ROUNDS=new Array(MathContext.prototype.ROUND_HALF_UP,MathContext.prototype.ROUND_UNNECESSARY,MathContext.prototype.ROUND_CEILING,MathContext.prototype.ROUND_DOWN,MathContext.prototype.ROUND_FLOOR,MathContext.prototype.ROUND_HALF_DOWN,MathContext.prototype.ROUND_HALF_EVEN,MathContext.prototype.ROUND_UP);


 //--private static final java.lang.String ROUNDWORDS[]=new java.lang.String[]{"ROUND_HALF_UP","ROUND_UNNECESSARY","ROUND_CEILING","ROUND_DOWN","ROUND_FLOOR","ROUND_HALF_DOWN","ROUND_HALF_EVEN","ROUND_UP"}; // matching names of the ROUNDS values
 MathContext.prototype.ROUNDWORDS=new Array("ROUND_HALF_UP","ROUND_UNNECESSARY","ROUND_CEILING","ROUND_DOWN","ROUND_FLOOR","ROUND_HALF_DOWN","ROUND_HALF_EVEN","ROUND_UP"); // matching names of the ROUNDS values




 /* properties private constant unused */

 // Serialization version
 //--private static final long serialVersionUID=7163376998892515376L;

 /* properties public constant */
 /**
  * A <code>MathContext</code> object initialized to the default
  * settings for general-purpose arithmetic.  That is,
  * <code>digits=9 form=SCIENTIFIC lostDigits=false
  * roundingMode=ROUND_HALF_UP</code>.
  *
  * @see #SCIENTIFIC
  * @see #ROUND_HALF_UP
  * @stable ICU 2.0
  */
 //--public static final com.ibm.icu.math.MathContext DEFAULT=new com.ibm.icu.math.MathContext(DEFAULT_DIGITS,DEFAULT_FORM,DEFAULT_LOSTDIGITS,DEFAULT_ROUNDINGMODE);
 MathContext.prototype.DEFAULT=new MathContext(MathContext.prototype.DEFAULT_DIGITS,MathContext.prototype.DEFAULT_FORM,MathContext.prototype.DEFAULT_LOSTDIGITS,MathContext.prototype.DEFAULT_ROUNDINGMODE);




 /* ----- Constructors ----- */

 /**
  * Constructs a new <code>MathContext</code> with a specified
  * precision.
  * The other settings are set to the default values
  * (see {@link #DEFAULT}).
  *
  * An <code>IllegalArgumentException</code> is thrown if the
  * <code>setdigits</code> parameter is out of range
  * (&lt;0 or &gt;999999999).
  *
  * @param setdigits     The <code>int</code> digits setting
  *                      for this <code>MathContext</code>.
  * @throws IllegalArgumentException parameter out of range.
  * @stable ICU 2.0
  */

 //--public MathContext(int setdigits){
 //-- this(setdigits,DEFAULT_FORM,DEFAULT_LOSTDIGITS,DEFAULT_ROUNDINGMODE);
 //-- return;}


 /**
  * Constructs a new <code>MathContext</code> with a specified
  * precision and form.
  * The other settings are set to the default values
  * (see {@link #DEFAULT}).
  *
  * An <code>IllegalArgumentException</code> is thrown if the
  * <code>setdigits</code> parameter is out of range
  * (&lt;0 or &gt;999999999), or if the value given for the
  * <code>setform</code> parameter is not one of the appropriate
  * constants.
  *
  * @param setdigits     The <code>int</code> digits setting
  *                      for this <code>MathContext</code>.
  * @param setform       The <code>int</code> form setting
  *                      for this <code>MathContext</code>.
  * @throws IllegalArgumentException parameter out of range.
  * @stable ICU 2.0
  */

 //--public MathContext(int setdigits,int setform){
 //-- this(setdigits,setform,DEFAULT_LOSTDIGITS,DEFAULT_ROUNDINGMODE);
 //-- return;}

 /**
  * Constructs a new <code>MathContext</code> with a specified
  * precision, form, and lostDigits setting.
  * The roundingMode setting is set to its default value
  * (see {@link #DEFAULT}).
  *
  * An <code>IllegalArgumentException</code> is thrown if the
  * <code>setdigits</code> parameter is out of range
  * (&lt;0 or &gt;999999999), or if the value given for the
  * <code>setform</code> parameter is not one of the appropriate
  * constants.
  *
  * @param setdigits     The <code>int</code> digits setting
  *                      for this <code>MathContext</code>.
  * @param setform       The <code>int</code> form setting
  *                      for this <code>MathContext</code>.
  * @param setlostdigits The <code>boolean</code> lostDigits
  *                      setting for this <code>MathContext</code>.
  * @throws IllegalArgumentException parameter out of range.
  * @stable ICU 2.0
  */

 //--public MathContext(int setdigits,int setform,boolean setlostdigits){
 //-- this(setdigits,setform,setlostdigits,DEFAULT_ROUNDINGMODE);
 //-- return;}

 /**
  * Constructs a new <code>MathContext</code> with a specified
  * precision, form, lostDigits, and roundingMode setting.
  *
  * An <code>IllegalArgumentException</code> is thrown if the
  * <code>setdigits</code> parameter is out of range
  * (&lt;0 or &gt;999999999), or if the value given for the
  * <code>setform</code> or <code>setroundingmode</code> parameters is
  * not one of the appropriate constants.
  *
  * @param setdigits       The <code>int</code> digits setting
  *                        for this <code>MathContext</code>.
  * @param setform         The <code>int</code> form setting
  *                        for this <code>MathContext</code>.
  * @param setlostdigits   The <code>boolean</code> lostDigits
  *                        setting for this <code>MathContext</code>.
  * @param setroundingmode The <code>int</code> roundingMode setting
  *                        for this <code>MathContext</code>.
  * @throws IllegalArgumentException parameter out of range.
  * @stable ICU 2.0
  */

 //--public MathContext(int setdigits,int setform,boolean setlostdigits,int setroundingmode){super();
 function MathContext() {
  //-- members
  this.digits = 0;
  this.form = 0; // values for this must fit in a byte
  this.lostDigits = false;
  this.roundingMode = 0;

  //-- overloaded ctor
  var setform = this.DEFAULT_FORM;
  var setlostdigits = this.DEFAULT_LOSTDIGITS;
  var setroundingmode = this.DEFAULT_ROUNDINGMODE;
  if (MathContext.arguments.length == 4)
   {
    setform = MathContext.arguments[1];
    setlostdigits = MathContext.arguments[2];
    setroundingmode = MathContext.arguments[3];
   }
  else if (MathContext.arguments.length == 3)
   {
    setform = MathContext.arguments[1];
    setlostdigits = MathContext.arguments[2];
   }
  else if (MathContext.arguments.length == 2)
   {
    setform = MathContext.arguments[1];
   }
  else if (MathContext.arguments.length != 1)
   {
    throw "MathContext(): " + MathContext.arguments.length + " arguments given; expected 1 to 4";
   }
  var setdigits = MathContext.arguments[0];


  // set values, after checking
  if (setdigits!=this.DEFAULT_DIGITS)
   {
    if (setdigits<this.MIN_DIGITS)
     throw "MathContext(): Digits too small: "+setdigits;
    if (setdigits>this.MAX_DIGITS)
     throw "MathContext(): Digits too large: "+setdigits;
   }
  {/*select*/
  if (setform==this.SCIENTIFIC)
   {} // [most common]
  else if (setform==this.ENGINEERING)
   {}
  else if (setform==this.PLAIN)
   {}
  else{
   throw "MathContext() Bad form value: "+setform;
  }
  }
  if ((!(this.isValidRound(setroundingmode))))
   throw "MathContext(): Bad roundingMode value: "+setroundingmode;
  this.digits=setdigits;
  this.form=setform;
  this.lostDigits=setlostdigits; // [no bad value possible]
  this.roundingMode=setroundingmode;
  return;}

 /**
  * Returns the digits setting.
  * This value is always non-negative.
  *
  * @return an <code>int</code> which is the value of the digits
  *         setting
  * @stable ICU 2.0
  */

 //--public int getDigits(){
 function getDigits() {
  return this.digits;
  }

 /**
  * Returns the form setting.
  * This will be one of
  * {@link #ENGINEERING},
  * {@link #PLAIN}, or
  * {@link #SCIENTIFIC}.
  *
  * @return an <code>int</code> which is the value of the form setting
  * @stable ICU 2.0
  */

 //--public int getForm(){
 function getForm() {
  return this.form;
  }

 /**
  * Returns the lostDigits setting.
  * This will be either <code>true</code> (enabled) or
  * <code>false</code> (disabled).
  *
  * @return a <code>boolean</code> which is the value of the lostDigits
  *           setting
  * @stable ICU 2.0
  */

 //--public boolean getLostDigits(){
 function getLostDigits() {
  return this.lostDigits;
  }

 /**
  * Returns the roundingMode setting.
  * This will be one of
  * {@link  #ROUND_CEILING},
  * {@link  #ROUND_DOWN},
  * {@link  #ROUND_FLOOR},
  * {@link  #ROUND_HALF_DOWN},
  * {@link  #ROUND_HALF_EVEN},
  * {@link  #ROUND_HALF_UP},
  * {@link  #ROUND_UNNECESSARY}, or
  * {@link  #ROUND_UP}.
  *
  * @return an <code>int</code> which is the value of the roundingMode
  *         setting
  * @stable ICU 2.0
  */

 //--public int getRoundingMode(){
 function getRoundingMode() {
  return this.roundingMode;
  }

 /** Returns the <code>MathContext</code> as a readable string.
  * The <code>String</code> returned represents the settings of the
  * <code>MathContext</code> object as four blank-delimited words
  * separated by a single blank and with no leading or trailing blanks,
  * as follows:
  * <ol>
  * <li>
  * <code>digits=</code>, immediately followed by
  * the value of the digits setting as a numeric word.
  * <li>
  * <code>form=</code>, immediately followed by
  * the value of the form setting as an uppercase word
  * (one of <code>SCIENTIFIC</code>, <code>PLAIN</code>, or
  * <code>ENGINEERING</code>).
  * <li>
  * <code>lostDigits=</code>, immediately followed by
  * the value of the lostDigits setting
  * (<code>1</code> if enabled, <code>0</code> if disabled).
  * <li>
  * <code>roundingMode=</code>, immediately followed by
  * the value of the roundingMode setting as a word.
  * This word will be the same as the name of the corresponding public
  * constant.
  * </ol>
  * <p>
  * For example:
  * <br><code>
  * digits=9 form=SCIENTIFIC lostDigits=0 roundingMode=ROUND_HALF_UP
  * </code>
  * <p>
  * Additional words may be appended to the result of
  * <code>toString</code> in the future if more properties are added
  * to the class.
  *
  * @return a <code>String</code> representing the context settings.
  * @stable ICU 2.0
  */

 //--public java.lang.String toString(){
 function toString() {
  //--java.lang.String formstr=null;
  var formstr=null;
  //--int r=0;
  var r=0;
  //--java.lang.String roundword=null;
  var roundword=null;
  {/*select*/
  if (this.form==this.SCIENTIFIC)
   formstr="SCIENTIFIC";
  else if (this.form==this.ENGINEERING)
   formstr="ENGINEERING";
  else{
   formstr="PLAIN";/* form=PLAIN */
  }
  }
  {var $1=this.ROUNDS.length;r=0;r:for(;$1>0;$1--,r++){
   if (this.roundingMode==this.ROUNDS[r])
    {
     roundword=this.ROUNDWORDS[r];
     break r;
    }
   }
  }/*r*/
  return "digits="+this.digits+" "+"form="+formstr+" "+"lostDigits="+(this.lostDigits?"1":"0")+" "+"roundingMode="+roundword;
  }


 /* <sgml> Test whether round is valid. </sgml> */
 // This could be made shared for use by BigDecimal for setScale.

 //--private static boolean isValidRound(int testround){
 function isValidRound(testround) {
  //--int r=0;
  var r=0;
  {var $2=this.ROUNDS.length;r=0;r:for(;$2>0;$2--,r++){
   if (testround==this.ROUNDS[r])
    return true;
   }
  }/*r*/
  return false;
  }
return MathContext;
})();

var BigDecimal = (function (MathContext) {
/* Generated from 'BigDecimal.nrx' 8 Sep 2000 11:10:50 [v2.00] */
/* Options: Binary Comments Crossref Format Java Logo Strictargs Strictcase Trace2 Verbose3 */
//--package com.ibm.icu.math;
//--import java.math.BigInteger;
//--import com.ibm.icu.impl.Utility;

/* ------------------------------------------------------------------ */
/* BigDecimal -- Decimal arithmetic for Java                          */
/* ------------------------------------------------------------------ */
/* Copyright IBM Corporation, 1996, 2000.  All Rights Reserved.       */
/*                                                                    */
/* The BigDecimal class provides immutable arbitrary-precision        */
/* floating point (including integer) decimal numbers.                */
/*                                                                    */
/* As the numbers are decimal, there is an exact correspondence       */
/* between an instance of a BigDecimal object and its String          */
/* representation; the BigDecimal class provides direct conversions   */
/* to and from String and character array objects, and well as        */
/* conversions to and from the Java primitive types (which may not    */
/* be exact).                                                         */
/* ------------------------------------------------------------------ */
/* Notes:                                                             */
/*                                                                    */
/* 1. A BigDecimal object is never changed in value once constructed; */
/*    this avoids the need for locking.  Note in particular that the  */
/*    mantissa array may be shared between many BigDecimal objects,   */
/*    so that once exposed it must not be altered.                    */
/*                                                                    */
/* 2. This class looks at MathContext class fields directly (for      */
/*    performance).  It must not and does not change them.            */
/*                                                                    */
/* 3. Exponent checking is delayed until finish(), as we know         */
/*    intermediate calculations cannot cause 31-bit overflow.         */
/*    [This assertion depends on MAX_DIGITS in MathContext.]          */
/*                                                                    */
/* 4. Comments for the public API now follow the javadoc conventions. */
/*    The NetRexx -comments option is used to pass these comments     */
/*    through to the generated Java code (with -format, if desired).  */
/*                                                                    */
/* 5. System.arraycopy is faster than explicit loop as follows        */
/*      Mean length 4:  equal                                         */
/*      Mean length 8:  x2                                            */
/*      Mean length 16: x3                                            */
/*      Mean length 24: x4                                            */
/*    From prior experience, we expect mean length a little below 8,  */
/*    but arraycopy is still the one to use, in general, until later  */
/*    measurements suggest otherwise.                                 */
/*                                                                    */
/* 6. 'DMSRCN' referred to below is the original (1981) IBM S/370     */
/*    assembler code implementation of the algorithms below; it is    */
/*    now called IXXRCN and is available with the OS/390 and VM/ESA   */
/*    operating systems.                                              */
/* ------------------------------------------------------------------ */
/* Change History:                                                    */
/* 1997.09.02 Initial version (derived from netrexx.lang classes)     */
/* 1997.09.12 Add lostDigits checking                                 */
/* 1997.10.06 Change mantissa to a byte array                         */
/* 1997.11.22 Rework power [did not prepare arguments, etc.]          */
/* 1997.12.13 multiply did not prepare arguments                      */
/* 1997.12.14 add did not prepare and align arguments correctly       */
/* 1998.05.02 0.07 packaging changes suggested by Sun and Oracle      */
/* 1998.05.21 adjust remainder operator finalization                  */
/* 1998.06.04 rework to pass MathContext to finish() and round()      */
/* 1998.06.06 change format to use round(); support rounding modes    */
/* 1998.06.25 rename to BigDecimal and begin merge                    */
/*            zero can now have trailing zeros (i.e., exp\=0)         */
/* 1998.06.28 new methods: movePointXxxx, scale, toBigInteger         */
/*                         unscaledValue, valueof                     */
/* 1998.07.01 improve byteaddsub to allow array reuse, etc.           */
/* 1998.07.01 make null testing explicit to avoid JIT bug [Win32]     */
/* 1998.07.07 scaled division  [divide(BigDecimal, int, int)]         */
/* 1998.07.08 setScale, faster equals                                 */
/* 1998.07.11 allow 1E6 (no sign) <sigh>; new double/float conversion */
/* 1998.10.12 change package to com.ibm.icu.math                          */
/* 1998.12.14 power operator no longer rounds RHS [to match ANSI]     */
/*            add toBigDecimal() and BigDecimal(java.math.BigDecimal) */
/* 1998.12.29 improve byteaddsub by using table lookup                */
/* 1999.02.04 lostdigits=0 behaviour rounds instead of digits+1 guard */
/* 1999.02.05 cleaner code for BigDecimal(char[])                     */
/* 1999.02.06 add javadoc comments                                    */
/* 1999.02.11 format() changed from 7 to 2 method form                */
/* 1999.03.05 null pointer checking is no longer explicit             */
/* 1999.03.05 simplify; changes from discussion with J. Bloch:        */
/*            null no longer permitted for MathContext; drop boolean, */
/*            byte, char, float, short constructor, deprecate double  */
/*            constructor, no blanks in string constructor, add       */
/*            offset and length version of char[] constructor;        */
/*            add valueOf(double); drop booleanValue, charValue;      */
/*            add ...Exact versions of remaining convertors           */
/* 1999.03.13 add toBigIntegerExact                                   */
/* 1999.03.13 1.00 release to IBM Centre for Java Technology          */
/* 1999.05.27 1.01 correct 0-0.2 bug under scaled arithmetic          */
/* 1999.06.29 1.02 constructors should not allow exponent > 9 digits  */
/* 1999.07.03 1.03 lost digits should not be checked if digits=0      */
/* 1999.07.06      lost digits Exception message changed              */
/* 1999.07.10 1.04 more work on 0-0.2 (scaled arithmetic)             */
/* 1999.07.17      improve messages from pow method                   */
/* 1999.08.08      performance tweaks                                 */
/* 1999.08.15      fastpath in multiply                               */
/* 1999.11.05 1.05 fix problem in intValueExact [e.g., 5555555555]    */
/* 1999.12.22 1.06 remove multiply fastpath, and improve performance  */
/* 2000.01.01      copyright update [Y2K has arrived]                 */
/* 2000.06.18 1.08 no longer deprecate BigDecimal(double)             */
/* ------------------------------------------------------------------ */


/* JavaScript conversion (c) 2003 STZ-IDA and PTV AG, Karlsruhe, Germany */



function div(a, b) {
    return (a-(a%b))/b;
}

BigDecimal.prototype.div = div;

function arraycopy(src, srcindex, dest, destindex, length) {
    var i;
    if (destindex > srcindex) {
        // in case src and dest are equals, but also doesn't hurt
        // if they are different
        for (i = length-1; i >= 0; --i) {
            dest[i+destindex] = src[i+srcindex];
        }
    } else {
        for (i = 0; i < length; ++i) {
            dest[i+destindex] = src[i+srcindex];
        }
    }
}

BigDecimal.prototype.arraycopy = arraycopy;

function createArrayWithZeros(length) {
    var retVal = new Array(length);
    var i;
    for (i = 0; i < length; ++i) {
        retVal[i] = 0;
    }
    return retVal;
}

BigDecimal.prototype.createArrayWithZeros = createArrayWithZeros;


/**
 * The <code>BigDecimal</code> class implements immutable
 * arbitrary-precision decimal numbers.  The methods of the
 * <code>BigDecimal</code> class provide operations for fixed and
 * floating point arithmetic, comparison, format conversions, and
 * hashing.
 * <p>
 * As the numbers are decimal, there is an exact correspondence between
 * an instance of a <code>BigDecimal</code> object and its
 * <code>String</code> representation; the <code>BigDecimal</code> class
 * provides direct conversions to and from <code>String</code> and
 * character array (<code>char[]</code>) objects, as well as conversions
 * to and from the Java primitive types (which may not be exact) and
 * <code>BigInteger</code>.
 * <p>
 * In the descriptions of constructors and methods in this documentation,
 * the value of a <code>BigDecimal</code> number object is shown as the
 * result of invoking the <code>toString()</code> method on the object.
 * The internal representation of a decimal number is neither defined
 * nor exposed, and is not permitted to affect the result of any
 * operation.
 * <p>
 * The floating point arithmetic provided by this class is defined by
 * the ANSI X3.274-1996 standard, and is also documented at
 * <code>http://www2.hursley.ibm.com/decimal</code>
 * <br><i>[This URL will change.]</i>
 *
 * <h3>Operator methods</h3>
 * <p>
 * Operations on <code>BigDecimal</code> numbers are controlled by a
 * {@link MathContext} object, which provides the context (precision and
 * other information) for the operation. Methods that can take a
 * <code>MathContext</code> parameter implement the standard arithmetic
 * operators for <code>BigDecimal</code> objects and are known as
 * <i>operator methods</i>.  The default settings provided by the
 * constant {@link MathContext#DEFAULT} (<code>digits=9,
 * form=SCIENTIFIC, lostDigits=false, roundingMode=ROUND_HALF_UP</code>)
 * perform general-purpose floating point arithmetic to nine digits of
 * precision.  The <code>MathContext</code> parameter must not be
 * <code>null</code>.
 * <p>
 * Each operator method also has a version provided which does
 * not take a <code>MathContext</code> parameter.  For this version of
 * each method, the context settings used are <code>digits=0,
 * form=PLAIN, lostDigits=false, roundingMode=ROUND_HALF_UP</code>;
 * these settings perform fixed point arithmetic with unlimited
 * precision, as defined for the original BigDecimal class in Java 1.1
 * and Java 1.2.
 * <p>
 * For monadic operators, only the optional <code>MathContext</code>
 * parameter is present; the operation acts upon the current object.
 * <p>
 * For dyadic operators, a <code>BigDecimal</code> parameter is always
 * present; it must not be <code>null</code>.
 * The operation acts with the current object being the left-hand operand
 * and the <code>BigDecimal</code> parameter being the right-hand operand.
 * <p>
 * For example, adding two <code>BigDecimal</code> objects referred to
 * by the names <code>award</code> and <code>extra</code> could be
 * written as any of:
 * <p><code>
 *     award.add(extra)
 * <br>award.add(extra, MathContext.DEFAULT)
 * <br>award.add(extra, acontext)
 * </code>
 * <p>
 * (where <code>acontext</code> is a <code>MathContext</code> object),
 * which would return a <code>BigDecimal</code> object whose value is
 * the result of adding <code>award</code> and <code>extra</code> under
 * the appropriate context settings.
 * <p>
 * When a <code>BigDecimal</code> operator method is used, a set of
 * rules define what the result will be (and, by implication, how the
 * result would be represented as a character string).
 * These rules are defined in the BigDecimal arithmetic documentation
 * (see the URL above), but in summary:
 * <ul>
 * <li>Results are normally calculated with up to some maximum number of
 * significant digits.
 * For example, if the <code>MathContext</code> parameter for an operation
 * were <code>MathContext.DEFAULT</code> then the result would be
 * rounded to 9 digits; the division of 2 by 3 would then result in
 * 0.666666667.
 * <br>
 * You can change the default of 9 significant digits by providing the
 * method with a suitable <code>MathContext</code> object. This lets you
 * calculate using as many digits as you need -- thousands, if necessary.
 * Fixed point (scaled) arithmetic is indicated by using a
 * <code>digits</code> setting of 0 (or omitting the
 * <code>MathContext</code> parameter).
 * <br>
 * Similarly, you can change the algorithm used for rounding from the
 * default "classic" algorithm.
 * <li>
 * In standard arithmetic (that is, when the <code>form</code> setting
 * is not <code>PLAIN</code>), a zero result is always expressed as the
 * single digit <code>'0'</code> (that is, with no sign, decimal point,
 * or exponent part).
 * <li>
 * Except for the division and power operators in standard arithmetic,
 * trailing zeros are preserved (this is in contrast to binary floating
 * point operations and most electronic calculators, which lose the
 * information about trailing zeros in the fractional part of results).
 * <br>
 * So, for example:
 * <p><code>
 *     new BigDecimal("2.40").add(     new BigDecimal("2"))      =&gt; "4.40"
 * <br>new BigDecimal("2.40").subtract(new BigDecimal("2"))      =&gt; "0.40"
 * <br>new BigDecimal("2.40").multiply(new BigDecimal("2"))      =&gt; "4.80"
 * <br>new BigDecimal("2.40").divide(  new BigDecimal("2"), def) =&gt; "1.2"
 * </code>
 * <p>where the value on the right of the <code>=&gt;</code> would be the
 * result of the operation, expressed as a <code>String</code>, and
 * <code>def</code> (in this and following examples) refers to
 * <code>MathContext.DEFAULT</code>).
 * This preservation of trailing zeros is desirable for most
 * calculations (including financial calculations).
 * If necessary, trailing zeros may be easily removed using division by 1.
 * <li>
 * In standard arithmetic, exponential form is used for a result
 * depending on its value and the current setting of <code>digits</code>
 * (the default is 9 digits).
 * If the number of places needed before the decimal point exceeds the
 * <code>digits</code> setting, or the absolute value of the number is
 * less than <code>0.000001</code>, then the number will be expressed in
 * exponential notation; thus
 * <p><code>
 *   new BigDecimal("1e+6").multiply(new BigDecimal("1e+6"), def)
 * </code>
 * <p>results in <code>1E+12</code> instead of
 * <code>1000000000000</code>, and
 * <p><code>
 *   new BigDecimal("1").divide(new BigDecimal("3E+10"), def)
 * </code>
 * <p>results in <code>3.33333333E-11</code> instead of
 * <code>0.0000000000333333333</code>.
 * <p>
 * The form of the exponential notation (scientific or engineering) is
 * determined by the <code>form</code> setting.
 * <eul>
 * <p>
 * The names of methods in this class follow the conventions established
 * by <code>java.lang.Number</code>, <code>java.math.BigInteger</code>,
 * and <code>java.math.BigDecimal</code> in Java 1.1 and Java 1.2.
 *
 * @see     MathContext
 * @author  Mike Cowlishaw
 * @stable ICU 2.0
 */

//--public class BigDecimal extends java.lang.Number implements java.io.Serializable,java.lang.Comparable{
//-- private static final java.lang.String $0="BigDecimal.nrx";

 //-- methods
 BigDecimal.prototype.abs = abs;
 BigDecimal.prototype.add = add;
 BigDecimal.prototype.compareTo = compareTo;
 BigDecimal.prototype.divide = divide;
 BigDecimal.prototype.divideInteger = divideInteger;
 BigDecimal.prototype.max = max;
 BigDecimal.prototype.min = min;
 BigDecimal.prototype.multiply = multiply;
 BigDecimal.prototype.negate = negate;
 BigDecimal.prototype.plus = plus;
 BigDecimal.prototype.pow = pow;
 BigDecimal.prototype.remainder = remainder;
 BigDecimal.prototype.subtract = subtract;
 BigDecimal.prototype.equals = equals;
 BigDecimal.prototype.format = format;
 BigDecimal.prototype.intValueExact = intValueExact;
 BigDecimal.prototype.movePointLeft = movePointLeft;
 BigDecimal.prototype.movePointRight = movePointRight;
 BigDecimal.prototype.scale = scale;
 BigDecimal.prototype.setScale = setScale;
 BigDecimal.prototype.signum = signum;
 BigDecimal.prototype.toString = toString;
 BigDecimal.prototype.layout = layout;
 BigDecimal.prototype.intcheck = intcheck;
 BigDecimal.prototype.dodivide = dodivide;
 BigDecimal.prototype.bad = bad;
 BigDecimal.prototype.badarg = badarg;
 BigDecimal.prototype.extend = extend;
 BigDecimal.prototype.byteaddsub = byteaddsub;
 BigDecimal.prototype.diginit = diginit;
 BigDecimal.prototype.clone = clone;
 BigDecimal.prototype.checkdigits = checkdigits;
 BigDecimal.prototype.round = round;
 BigDecimal.prototype.allzero = allzero;
 BigDecimal.prototype.finish = finish;

 // Convenience methods
 BigDecimal.prototype.isGreaterThan = isGreaterThan;
 BigDecimal.prototype.isLessThan = isLessThan;
 BigDecimal.prototype.isGreaterThanOrEqualTo = isGreaterThanOrEqualTo;
 BigDecimal.prototype.isLessThanOrEqualTo = isLessThanOrEqualTo;
 BigDecimal.prototype.isPositive = isPositive;
 BigDecimal.prototype.isNegative = isNegative;
 BigDecimal.prototype.isZero = isZero;


 /* ----- Constants ----- */
 /* properties constant public */ // useful to others
 // the rounding modes (copied here for upwards compatibility)
 /**
  * Rounding mode to round to a more positive number.
  * @see MathContext#ROUND_CEILING
  * @stable ICU 2.0
  */
 //--public static final int ROUND_CEILING=com.ibm.icu.math.MathContext.ROUND_CEILING;
 BigDecimal.ROUND_CEILING = BigDecimal.prototype.ROUND_CEILING = MathContext.prototype.ROUND_CEILING;

 /**
  * Rounding mode to round towards zero.
  * @see MathContext#ROUND_DOWN
  * @stable ICU 2.0
  */
 //--public static final int ROUND_DOWN=com.ibm.icu.math.MathContext.ROUND_DOWN;
 BigDecimal.ROUND_DOWN = BigDecimal.prototype.ROUND_DOWN = MathContext.prototype.ROUND_DOWN;

 /**
  * Rounding mode to round to a more negative number.
  * @see MathContext#ROUND_FLOOR
  * @stable ICU 2.0
  */
 //--public static final int ROUND_FLOOR=com.ibm.icu.math.MathContext.ROUND_FLOOR;
 BigDecimal.ROUND_FLOOR = BigDecimal.prototype.ROUND_FLOOR = MathContext.prototype.ROUND_FLOOR;

 /**
  * Rounding mode to round to nearest neighbor, where an equidistant
  * value is rounded down.
  * @see MathContext#ROUND_HALF_DOWN
  * @stable ICU 2.0
  */
 //--public static final int ROUND_HALF_DOWN=com.ibm.icu.math.MathContext.ROUND_HALF_DOWN;
 BigDecimal.ROUND_HALF_DOWN = BigDecimal.prototype.ROUND_HALF_DOWN = MathContext.prototype.ROUND_HALF_DOWN;

 /**
  * Rounding mode to round to nearest neighbor, where an equidistant
  * value is rounded to the nearest even neighbor.
  * @see MathContext#ROUND_HALF_EVEN
  * @stable ICU 2.0
  */
 //--public static final int ROUND_HALF_EVEN=com.ibm.icu.math.MathContext.ROUND_HALF_EVEN;
 BigDecimal.ROUND_HALF_EVEN = BigDecimal.prototype.ROUND_HALF_EVEN = MathContext.prototype.ROUND_HALF_EVEN;

 /**
  * Rounding mode to round to nearest neighbor, where an equidistant
  * value is rounded up.
  * @see MathContext#ROUND_HALF_UP
  * @stable ICU 2.0
  */
 //--public static final int ROUND_HALF_UP=com.ibm.icu.math.MathContext.ROUND_HALF_UP;
 BigDecimal.ROUND_HALF_UP = BigDecimal.prototype.ROUND_HALF_UP = MathContext.prototype.ROUND_HALF_UP;

 /**
  * Rounding mode to assert that no rounding is necessary.
  * @see MathContext#ROUND_UNNECESSARY
  * @stable ICU 2.0
  */
 //--public static final int ROUND_UNNECESSARY=com.ibm.icu.math.MathContext.ROUND_UNNECESSARY;
 BigDecimal.ROUND_UNNECESSARY = BigDecimal.prototype.ROUND_UNNECESSARY = MathContext.prototype.ROUND_UNNECESSARY;

 /**
  * Rounding mode to round away from zero.
  * @see MathContext#ROUND_UP
  * @stable ICU 2.0
  */
 //--public static final int ROUND_UP=com.ibm.icu.math.MathContext.ROUND_UP;
 BigDecimal.ROUND_UP = BigDecimal.prototype.ROUND_UP = MathContext.prototype.ROUND_UP;

 /* properties constant private */ // locals
 //--private static final byte ispos=1; // ind: indicates positive (must be 1)
 //--private static final byte iszero=0; // ind: indicates zero     (must be 0)
 //--private static final byte isneg=-1; // ind: indicates negative (must be -1)
 BigDecimal.prototype.ispos = 1;
 BigDecimal.prototype.iszero = 0;
 BigDecimal.prototype.isneg = -1;
 // [later could add NaN, +/- infinity, here]

 //--private static final int MinExp=-999999999; // minimum exponent allowed
 //--private static final int MaxExp=999999999; // maximum exponent allowed
 //--private static final int MinArg=-999999999; // minimum argument integer
 //--private static final int MaxArg=999999999; // maximum argument integer
 BigDecimal.prototype.MinExp=-999999999; // minimum exponent allowed
 BigDecimal.prototype.MaxExp=999999999; // maximum exponent allowed
 BigDecimal.prototype.MinArg=-999999999; // minimum argument integer
 BigDecimal.prototype.MaxArg=999999999; // maximum argument integer

 //--private static final com.ibm.icu.math.MathContext plainMC=new com.ibm.icu.math.MathContext(0,com.ibm.icu.math.MathContext.PLAIN); // context for plain unlimited math
 BigDecimal.prototype.plainMC=new MathContext(0, MathContext.prototype.PLAIN);

 /* properties constant private unused */ // present but not referenced

 // Serialization version
 //--private static final long serialVersionUID=8245355804974198832L;

 //--private static final java.lang.String copyright=" Copyright (c) IBM Corporation 1996, 2000.  All rights reserved. ";

 /* properties static private */
 // Precalculated constant arrays (used by byteaddsub)
 //--private static byte bytecar[]=new byte[(90+99)+1]; // carry/borrow array
 //--private static byte bytedig[]=diginit(); // next digit array
 BigDecimal.prototype.bytecar = new Array((90+99)+1);
 BigDecimal.prototype.bytedig = diginit();

 /**
  * The <code>BigDecimal</code> constant "0".
  *
  * @see #ONE
  * @see #TEN
  * @stable ICU 2.0
  */
 //--public static final com.ibm.icu.math.BigDecimal ZERO=new com.ibm.icu.math.BigDecimal((long)0); // use long as we want the int constructor
 // .. to be able to use this, for speed
BigDecimal.ZERO = BigDecimal.prototype.ZERO = new BigDecimal("0");

 /**
  * The <code>BigDecimal</code> constant "1".
  *
  * @see #TEN
  * @see #ZERO
  * @stable ICU 2.0
  */
 //--public static final com.ibm.icu.math.BigDecimal ONE=new com.ibm.icu.math.BigDecimal((long)1); // use long as we want the int constructor
 // .. to be able to use this, for speed
BigDecimal.ONE = BigDecimal.prototype.ONE = new BigDecimal("1");

 /**
  * The <code>BigDecimal</code> constant "10".
  *
  * @see #ONE
  * @see #ZERO
  * @stable ICU 2.0
  */
 //--public static final com.ibm.icu.math.BigDecimal TEN=new com.ibm.icu.math.BigDecimal(10);
 BigDecimal.TEN = BigDecimal.prototype.TEN = new BigDecimal("10");

 /* ----- Instance properties [all private and immutable] ----- */
 /* properties private */

 /**
  * The indicator. This may take the values:
  * <ul>
  * <li>ispos  -- the number is positive
  * <li>iszero -- the number is zero
  * <li>isneg  -- the number is negative
  * </ul>
  *
  * @serial
  */
 //--private byte ind; // assumed undefined
 // Note: some code below assumes IND = Sign [-1, 0, 1], at present.
 // We only need two bits for this, but use a byte [also permits
 // smooth future extension].

 /**
  * The formatting style. This may take the values:
  * <ul>
  * <li>MathContext.PLAIN        -- no exponent needed
  * <li>MathContext.SCIENTIFIC   -- scientific notation required
  * <li>MathContext.ENGINEERING  -- engineering notation required
  * </ul>
  * <p>
  * This property is an optimization; it allows us to defer number
  * layout until it is actually needed as a string, hence avoiding
  * unnecessary formatting.
  *
  * @serial
  */
 //--private byte form=(byte)com.ibm.icu.math.MathContext.PLAIN; // assumed PLAIN
 // We only need two bits for this, at present, but use a byte
 // [again, to allow for smooth future extension]

 /**
  * The value of the mantissa.
  * <p>
  * Once constructed, this may become shared between several BigDecimal
  * objects, so must not be altered.
  * <p>
  * For efficiency (speed), this is a byte array, with each byte
  * taking a value of 0 -> 9.
  * <p>
  * If the first byte is 0 then the value of the number is zero (and
  * mant.length=1, except when constructed from a plain number, for
  * example, 0.000).
  *
  * @serial
  */
 //--private byte mant[]; // assumed null

 /**
  * The exponent.
  * <p>
  * For fixed point arithmetic, scale is <code>-exp</code>, and can
  * apply to zero.
  *
  * Note that this property can have a value less than MinExp when
  * the mantissa has more than one digit.
  *
  * @serial
  */
 //--private int exp;
 // assumed 0

 /* ---------------------------------------------------------------- */
 /* Constructors                                                     */
 /* ---------------------------------------------------------------- */

 /**
  * Constructs a <code>BigDecimal</code> object from a
  * <code>java.math.BigDecimal</code>.
  * <p>
  * Constructs a <code>BigDecimal</code> as though the parameter had
  * been represented as a <code>String</code> (using its
  * <code>toString</code> method) and the
  * {@link #BigDecimal(java.lang.String)} constructor had then been
  * used.
  * The parameter must not be <code>null</code>.
  * <p>
  * <i>(Note: this constructor is provided only in the
  * <code>com.ibm.icu.math</code> version of the BigDecimal class.
  * It would not be present in a <code>java.math</code> version.)</i>
  *
  * @param bd The <code>BigDecimal</code> to be translated.
  * @stable ICU 2.0
  */

 //--public BigDecimal(java.math.BigDecimal bd){
 //-- this(bd.toString());
 //-- return;}

 /**
  * Constructs a <code>BigDecimal</code> object from a
  * <code>BigInteger</code>, with scale 0.
  * <p>
  * Constructs a <code>BigDecimal</code> which is the exact decimal
  * representation of the <code>BigInteger</code>, with a scale of
  * zero.
  * The value of the <code>BigDecimal</code> is identical to the value
  * of the <code>BigInteger</code>.
  * The parameter must not be <code>null</code>.
  * <p>
  * The <code>BigDecimal</code> will contain only decimal digits,
  * prefixed with a leading minus sign (hyphen) if the
  * <code>BigInteger</code> is negative.  A leading zero will be
  * present only if the <code>BigInteger</code> is zero.
  *
  * @param bi The <code>BigInteger</code> to be converted.
  * @stable ICU 2.0
  */

 //--public BigDecimal(java.math.BigInteger bi){
 //-- this(bi.toString(10));
 //-- return;}
 // exp remains 0

 /**
  * Constructs a <code>BigDecimal</code> object from a
  * <code>BigInteger</code> and a scale.
  * <p>
  * Constructs a <code>BigDecimal</code> which is the exact decimal
  * representation of the <code>BigInteger</code>, scaled by the
  * second parameter, which may not be negative.
  * The value of the <code>BigDecimal</code> is the
  * <code>BigInteger</code> divided by ten to the power of the scale.
  * The <code>BigInteger</code> parameter must not be
  * <code>null</code>.
  * <p>
  * The <code>BigDecimal</code> will contain only decimal digits, (with
  * an embedded decimal point followed by <code>scale</code> decimal
  * digits if the scale is positive), prefixed with a leading minus
  * sign (hyphen) if the <code>BigInteger</code> is negative.  A
  * leading zero will be present only if the <code>BigInteger</code> is
  * zero.
  *
  * @param  bi    The <code>BigInteger</code> to be converted.
  * @param  scale The <code>int</code> specifying the scale.
  * @throws NumberFormatException if the scale is negative.
  * @stable ICU 2.0
  */

 //--public BigDecimal(java.math.BigInteger bi,int scale){
 //-- this(bi.toString(10));
 //-- if (scale<0)
 //--  throw new java.lang.NumberFormatException("Negative scale:"+" "+scale);
 //-- exp=(int)-scale; // exponent is -scale
 //-- return;}

 /**
  * Constructs a <code>BigDecimal</code> object from an array of characters.
  * <p>
  * Constructs a <code>BigDecimal</code> as though a
  * <code>String</code> had been constructed from the character array
  * and the {@link #BigDecimal(java.lang.String)} constructor had then
  * been used. The parameter must not be <code>null</code>.
  * <p>
  * Using this constructor is faster than using the
  * <code>BigDecimal(String)</code> constructor if the string is
  * already available in character array form.
  *
  * @param inchars The <code>char[]</code> array containing the number
  *                to be converted.
  * @throws NumberFormatException if the parameter is not a valid
  *                number.
  * @stable ICU 2.0
  */

 //--public BigDecimal(char inchars[]){
 //-- this(inchars,0,inchars.length);
 //-- return;}

 /**
  * Constructs a <code>BigDecimal</code> object from an array of characters.
  * <p>
  * Constructs a <code>BigDecimal</code> as though a
  * <code>String</code> had been constructed from the character array
  * (or a subarray of that array) and the
  * {@link #BigDecimal(java.lang.String)} constructor had then been
  * used. The first parameter must not be <code>null</code>, and the
  * subarray must be wholly contained within it.
  * <p>
  * Using this constructor is faster than using the
  * <code>BigDecimal(String)</code> constructor if the string is
  * already available within a character array.
  *
  * @param inchars The <code>char[]</code> array containing the number
  *                to be converted.
  * @param offset  The <code>int</code> offset into the array of the
  *                start of the number to be converted.
  * @param length  The <code>int</code> length of the number.
  * @throws NumberFormatException if the parameter is not a valid
  *                number for any reason.
  * @stable ICU 2.0
  */

 //--public BigDecimal(char inchars[],int offset,int length){super();
 function BigDecimal() {
  //-- members
  this.ind = 0;
  this.form = MathContext.prototype.PLAIN;
  this.mant = null;
  this.exp = 0;

  //-- overloaded ctor
  if (BigDecimal.arguments.length == 0)
   return;
  var inchars;
  var offset;
  var length;
  if (BigDecimal.arguments.length == 1)
   {
    inchars = BigDecimal.arguments[0];
    offset = 0;
    length = inchars.length;
   }
  else
   {
    inchars = BigDecimal.arguments[0];
    offset = BigDecimal.arguments[1];
    length = BigDecimal.arguments[2];
   }
  if (typeof inchars == "string")
   {
    inchars = inchars.split("");
   }

  //--boolean exotic;
  var exotic;
  //--boolean hadexp;
  var hadexp;
  //--int d;
  var d;
  //--int dotoff;
  var dotoff;
  //--int last;
  var last;
  //--int i=0;
  var i=0;
  //--char si=0;
  var si=0;
  //--boolean eneg=false;
  var eneg=false;
  //--int k=0;
  var k=0;
  //--int elen=0;
  var elen=0;
  //--int j=0;
  var j=0;
  //--char sj=0;
  var sj=0;
  //--int dvalue=0;
  var dvalue=0;
  //--int mag=0;
  var mag=0;
  // This is the primary constructor; all incoming strings end up
  // here; it uses explicit (inline) parsing for speed and to avoid
  // generating intermediate (temporary) objects of any kind.
  // 1998.06.25: exponent form built only if E/e in string
  // 1998.06.25: trailing zeros not removed for zero
  // 1999.03.06: no embedded blanks; allow offset and length
  if (length<=0)
   this.bad("BigDecimal(): ", inchars); // bad conversion (empty string)
  // [bad offset will raise array bounds exception]

  /* Handle and step past sign */
  this.ind=this.ispos; // assume positive
  if (inchars[0]==('-'))
   {
    length--;
    if (length==0)
     this.bad("BigDecimal(): ", inchars); // nothing after sign
    this.ind=this.isneg;
    offset++;
   }
  else
   if (inchars[0]==('+'))
    {
     length--;
     if (length==0)
      this.bad("BigDecimal(): ", inchars); // nothing after sign
     offset++;
    }

  /* We're at the start of the number */
  exotic=false; // have extra digits
  hadexp=false; // had explicit exponent
  d=0; // count of digits found
  dotoff=-1; // offset where dot was found
  last=-1; // last character of mantissa
  {var $1=length;i=offset;i:for(;$1>0;$1--,i++){
   si=inchars[i];
   if (si>='0')  // test for Arabic digit
    if (si<='9')
     {
      last=i;
      d++; // still in mantissa
      continue i;
     }
   if (si=='.')
    { // record and ignore
     if (dotoff>=0)
      this.bad("BigDecimal(): ", inchars); // two dots
     dotoff=i-offset; // offset into mantissa
     continue i;
    }
   if (si!='e')
    if (si!='E')
     { // expect an extra digit
      if (si<'0' || si>'9')
       this.bad("BigDecimal(): ", inchars); // not a number
      // defer the base 10 check until later to avoid extra method call
      exotic=true; // will need conversion later
      last=i;
      d++; // still in mantissa
      continue i;
     }
   /* Found 'e' or 'E' -- now process explicit exponent */
   // 1998.07.11: sign no longer required
   if ((i-offset)>(length-2))
    this.bad("BigDecimal(): ", inchars); // no room for even one digit
   eneg=false;
   if ((inchars[i+1])==('-'))
    {
     eneg=true;
     k=i+2;
    }
   else
    if ((inchars[i+1])==('+'))
     k=i+2;
    else
     k=i+1;
   // k is offset of first expected digit
   elen=length-((k-offset)); // possible number of digits
   if ((elen==0)||(elen>9))
    this.bad("BigDecimal(): ", inchars); // 0 or more than 9 digits
   {var $2=elen;j=k;j:for(;$2>0;$2--,j++){
    sj=inchars[j];
    if (sj<'0')
     this.bad("BigDecimal(): ", inchars); // always bad
    if (sj>'9')
     { // maybe an exotic digit
      /*if (si<'0' || si>'9')
       this.bad(inchars); // not a number
      dvalue=java.lang.Character.digit(sj,10); // check base
      if (dvalue<0)
       bad(inchars); // not base 10*/
      this.bad("BigDecimal(): ", inchars);
     }
    else
     dvalue=sj-'0';
    this.exp=(this.exp*10)+dvalue;
    }
   }/*j*/
   if (eneg)
    this.exp=-this.exp; // was negative
   hadexp=true; // remember we had one
   break i; // we are done
   }
  }/*i*/

  /* Here when all inspected */
  if (d==0)
   this.bad("BigDecimal(): ", inchars); // no mantissa digits
  if (dotoff>=0)
   this.exp=(this.exp+dotoff)-d; // adjust exponent if had dot

  /* strip leading zeros/dot (leave final if all 0's) */
  {var $3=last-1;i=offset;i:for(;i<=$3;i++){
   si=inchars[i];
   if (si=='0')
    {
     offset++;
     dotoff--;
     d--;
    }
   else
    if (si=='.')
     {
      offset++; // step past dot
      dotoff--;
     }
    else
     if (si<='9')
      break i;/* non-0 */
     else
      {/* exotic */
       //if ((java.lang.Character.digit(si,10))!=0)
        break i; // non-0 or bad
       // is 0 .. strip like '0'
       //offset++;
       //dotoff--;
       //d--;
      }
   }
  }/*i*/

  /* Create the mantissa array */
  this.mant=new Array(d); // we know the length
  j=offset; // input offset
  if (exotic)
   {exotica:do{ // slow: check for exotica
    {var $4=d;i=0;i:for(;$4>0;$4--,i++){
     if (i==dotoff)
      j++; // at dot
     sj=inchars[j];
     if (sj<='9')
      this.mant[i]=sj-'0';/* easy */
     else
      {
       //dvalue=java.lang.Character.digit(sj,10);
       //if (dvalue<0)
        this.bad("BigDecimal(): ", inchars); // not a number after all
       //mant[i]=(byte)dvalue;
      }
     j++;
     }
    }/*i*/
   }while(false);}/*exotica*/
  else
   {simple:do{
    {var $5=d;i=0;i:for(;$5>0;$5--,i++){
     if (i==dotoff)
      j++;
     this.mant[i]=inchars[j]-'0';
     j++;
     }
    }/*i*/
   }while(false);}/*simple*/

  /* Looks good.  Set the sign indicator and form, as needed. */
  // Trailing zeros are preserved
  // The rule here for form is:
  //   If no E-notation, then request plain notation
  //   Otherwise act as though add(0,DEFAULT) and request scientific notation
  // [form is already PLAIN]
  if (this.mant[0]==0)
   {
    this.ind=this.iszero; // force to show zero
    // negative exponent is significant (e.g., -3 for 0.000) if plain
    if (this.exp>0)
     this.exp=0; // positive exponent can be ignored
    if (hadexp)
     { // zero becomes single digit from add
      this.mant=this.ZERO.mant;
      this.exp=0;
     }
   }
  else
   { // non-zero
    // [ind was set earlier]
    // now determine form
    if (hadexp)
     {
      this.form=MathContext.prototype.SCIENTIFIC;
      // 1999.06.29 check for overflow
      mag=(this.exp+this.mant.length)-1; // true exponent in scientific notation
      if ((mag<this.MinExp)||(mag>this.MaxExp))
       this.bad("BigDecimal(): ", inchars);
     }
   }
  // say 'BD(c[]): mant[0] mantlen exp ind form:' mant[0] mant.length exp ind form
  return;
  }

 /**
  * Constructs a <code>BigDecimal</code> object directly from a
  * <code>double</code>.
  * <p>
  * Constructs a <code>BigDecimal</code> which is the exact decimal
  * representation of the 64-bit signed binary floating point
  * parameter.
  * <p>
  * Note that this constructor it an exact conversion; it does not give
  * the same result as converting <code>num</code> to a
  * <code>String</code> using the <code>Double.toString()</code> method
  * and then using the {@link #BigDecimal(java.lang.String)}
  * constructor.
  * To get that result, use the static {@link #valueOf(double)}
  * method to construct a <code>BigDecimal</code> from a
  * <code>double</code>.
  *
  * @param num The <code>double</code> to be converted.
  * @throws NumberFormatException if the parameter is infinite or
  *            not a number.
  * @stable ICU 2.0
  */

 //--public BigDecimal(double num){
 //-- // 1999.03.06: use exactly the old algorithm
 //-- // 2000.01.01: note that this constructor does give an exact result,
 //-- //             so perhaps it should not be deprecated
 //-- // 2000.06.18: no longer deprecated
 //-- this((new java.math.BigDecimal(num)).toString());
 //-- return;}

 /**
  * Constructs a <code>BigDecimal</code> object directly from a
  * <code>int</code>.
  * <p>
  * Constructs a <code>BigDecimal</code> which is the exact decimal
  * representation of the 32-bit signed binary integer parameter.
  * The <code>BigDecimal</code> will contain only decimal digits,
  * prefixed with a leading minus sign (hyphen) if the parameter is
  * negative.
  * A leading zero will be present only if the parameter is zero.
  *
  * @param num The <code>int</code> to be converted.
  * @stable ICU 2.0
  */

 //--public BigDecimal(int num){super();
 //-- int mun;
 //-- int i=0;
 //-- // We fastpath commoners
 //-- if (num<=9)
 //--  if (num>=(-9))
 //--   {singledigit:do{
 //--    // very common single digit case
 //--    {/*select*/
 //--    if (num==0)
 //--     {
 //--      mant=ZERO.mant;
 //--      ind=iszero;
 //--     }
 //--    else if (num==1)
 //--     {
 //--      mant=ONE.mant;
 //--      ind=ispos;
 //--     }
 //--    else if (num==(-1))
 //--     {
 //--      mant=ONE.mant;
 //--      ind=isneg;
 //--     }
 //--    else{
 //--     {
 //--      mant=new byte[1];
 //--      if (num>0)
 //--       {
 //--        mant[0]=(byte)num;
 //--        ind=ispos;
 //--       }
 //--      else
 //--       { // num<-1
 //--        mant[0]=(byte)((int)-num);
 //--        ind=isneg;
 //--       }
 //--     }
 //--    }
 //--    }
 //--    return;
 //--   }while(false);}/*singledigit*/
 //--
 //-- /* We work on negative numbers so we handle the most negative number */
 //-- if (num>0)
 //--  {
 //--   ind=ispos;
 //--   num=(int)-num;
 //--  }
 //-- else
 //--  ind=isneg;/* negative */ // [0 case already handled]
 //-- // [it is quicker, here, to pre-calculate the length with
 //-- // one loop, then allocate exactly the right length of byte array,
 //-- // then re-fill it with another loop]
 //-- mun=num; // working copy
 //-- {i=9;i:for(;;i--){
 //--  mun=mun/10;
 //--  if (mun==0)
 //--   break i;
 //--  }
 //-- }/*i*/
 //-- // i is the position of the leftmost digit placed
 //-- mant=new byte[10-i];
 //-- {i=(10-i)-1;i:for(;;i--){
 //--  mant[i]=(byte)-(((byte)(num%10)));
 //--  num=num/10;
 //--  if (num==0)
 //--   break i;
 //--  }
 //-- }/*i*/
 //-- return;
 //-- }

 /**
  * Constructs a <code>BigDecimal</code> object directly from a
  * <code>long</code>.
  * <p>
  * Constructs a <code>BigDecimal</code> which is the exact decimal
  * representation of the 64-bit signed binary integer parameter.
  * The <code>BigDecimal</code> will contain only decimal digits,
  * prefixed with a leading minus sign (hyphen) if the parameter is
  * negative.
  * A leading zero will be present only if the parameter is zero.
  *
  * @param num The <code>long</code> to be converted.
  * @stable ICU 2.0
  */

 //--public BigDecimal(long num){super();
 //-- long mun;
 //-- int i=0;
 //-- // Not really worth fastpathing commoners in this constructor [also,
 //-- // we use this to construct the static constants].
 //-- // This is much faster than: this(String.valueOf(num).toCharArray())
 //-- /* We work on negative num so we handle the most negative number */
 //-- if (num>0)
 //--  {
 //--   ind=ispos;
 //--   num=(long)-num;
 //--  }
 //-- else
 //--  if (num==0)
 //--   ind=iszero;
 //--  else
 //--   ind=isneg;/* negative */
 //-- mun=num;
 //-- {i=18;i:for(;;i--){
 //--  mun=mun/10;
 //--  if (mun==0)
 //--   break i;
 //--  }
 //-- }/*i*/
 //-- // i is the position of the leftmost digit placed
 //-- mant=new byte[19-i];
 //-- {i=(19-i)-1;i:for(;;i--){
 //--  mant[i]=(byte)-(((byte)(num%10)));
 //--  num=num/10;
 //--  if (num==0)
 //--   break i;
 //--  }
 //-- }/*i*/
 //-- return;
 //-- }

 /**
  * Constructs a <code>BigDecimal</code> object from a <code>String</code>.
  * <p>
  * Constructs a <code>BigDecimal</code> from the parameter, which must
  * not be <code>null</code> and must represent a valid <i>number</i>,
  * as described formally in the documentation referred to
  * {@link BigDecimal above}.
  * <p>
  * In summary, numbers in <code>String</code> form must have at least
  * one digit, may have a leading sign, may have a decimal point, and
  * exponential notation may be used.  They follow conventional syntax,
  * and may not contain blanks.
  * <p>
  * Some valid strings from which a <code>BigDecimal</code> might
  * be constructed are:
  * <pre>
  *       "0"         -- Zero
  *      "12"         -- A whole number
  *     "-76"         -- A signed whole number
  *      "12.70"      -- Some decimal places
  *     "+0.003"      -- Plus sign is allowed
  *      "17."        -- The same as 17
  *        ".5"       -- The same as 0.5
  *      "4E+9"       -- Exponential notation
  *       "0.73e-7"   -- Exponential notation
  * </pre>
  * <p>
  * (Exponential notation means that the number includes an optional
  * sign and a power of ten following an '</code>E</code>' that
  * indicates how the decimal point will be shifted.  Thus the
  * <code>"4E+9"</code> above is just a short way of writing
  * <code>4000000000</code>, and the <code>"0.73e-7"</code> is short
  * for <code>0.000000073</code>.)
  * <p>
  * The <code>BigDecimal</code> constructed from the String is in a
  * standard form, with no blanks, as though the
  * {@link #add(BigDecimal)} method had been used to add zero to the
  * number with unlimited precision.
  * If the string uses exponential notation (that is, includes an
  * <code>e</code> or an <code>E</code>), then the
  * <code>BigDecimal</code> number will be expressed in scientific
  * notation (where the power of ten is adjusted so there is a single
  * non-zero digit to the left of the decimal point); in this case if
  * the number is zero then it will be expressed as the single digit 0,
  * and if non-zero it will have an exponent unless that exponent would
  * be 0.  The exponent must fit in nine digits both before and after it
  * is expressed in scientific notation.
  * <p>
  * Any digits in the parameter must be decimal; that is,
  * <code>Character.digit(c, 10)</code> (where </code>c</code> is the
  * character in question) would not return -1.
  *
  * @param string The <code>String</code> to be converted.
  * @throws NumberFormatException if the parameter is not a valid
  * number.
  * @stable ICU 2.0
  */

 //--public BigDecimal(java.lang.String string){
 //-- this(string.toCharArray(),0,string.length());
 //-- return;}

 /* <sgml> Make a default BigDecimal object for local use. </sgml> */

 //--private BigDecimal(){super();
 //-- return;
 //-- }

 /* ---------------------------------------------------------------- */
 /* Operator methods [methods which take a context parameter]        */
 /* ---------------------------------------------------------------- */

 /**
  * Returns a plain <code>BigDecimal</code> whose value is the absolute
  * value of this <code>BigDecimal</code>.
  * <p>
  * The same as {@link #abs(MathContext)}, where the context is
  * <code>new MathContext(0, MathContext.PLAIN)</code>.
  * <p>
  * The length of the decimal part (the scale) of the result will
  * be <code>this.scale()</code>
  *
  * @return A <code>BigDecimal</code> whose value is the absolute
  *         value of this <code>BigDecimal</code>.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal abs(){
 //- return this.abs(plainMC);
 //- }

 /**
  * Returns a <code>BigDecimal</code> whose value is the absolute value
  * of this <code>BigDecimal</code>.
  * <p>
  * If the current object is zero or positive, then the same result as
  * invoking the {@link #plus(MathContext)} method with the same
  * parameter is returned.
  * Otherwise, the same result as invoking the
  * {@link #negate(MathContext)} method with the same parameter is
  * returned.
  *
  * @param  set The <code>MathContext</code> arithmetic settings.
  * @return     A <code>BigDecimal</code> whose value is the absolute
  *             value of this <code>BigDecimal</code>.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal abs(com.ibm.icu.math.MathContext set){
 function abs() {
  var set;
  if (abs.arguments.length == 1)
   {
    set = abs.arguments[0];
   }
  else if (abs.arguments.length == 0)
   {
    set = this.plainMC;
   }
  else
   {
    throw "abs(): " + abs.arguments.length + " arguments given; expected 0 or 1";
   }
  if (this.ind==this.isneg)
   return this.negate(set);
  return this.plus(set);
  }

 /**
  * Returns a plain <code>BigDecimal</code> whose value is
  * <code>this+rhs</code>, using fixed point arithmetic.
  * <p>
  * The same as {@link #add(BigDecimal, MathContext)},
  * where the <code>BigDecimal</code> is <code>rhs</code>,
  * and the context is <code>new MathContext(0, MathContext.PLAIN)</code>.
  * <p>
  * The length of the decimal part (the scale) of the result will be
  * the maximum of the scales of the two operands.
  *
  * @param  rhs The <code>BigDecimal</code> for the right hand side of
  *             the addition.
  * @return     A <code>BigDecimal</code> whose value is
  *             <code>this+rhs</code>, using fixed point arithmetic.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal add(com.ibm.icu.math.BigDecimal rhs){
 //-- return this.add(rhs,plainMC);
 //-- }

 /**
  * Returns a <code>BigDecimal</code> whose value is <code>this+rhs</code>.
  * <p>
  * Implements the addition (<b><code>+</code></b>) operator
  * (as defined in the decimal documentation, see {@link BigDecimal
  * class header}),
  * and returns the result as a <code>BigDecimal</code> object.
  *
  * @param  rhs The <code>BigDecimal</code> for the right hand side of
  *             the addition.
  * @param  set The <code>MathContext</code> arithmetic settings.
  * @return     A <code>BigDecimal</code> whose value is
  *             <code>this+rhs</code>.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal add(com.ibm.icu.math.BigDecimal rhs,com.ibm.icu.math.MathContext set){
 function add() {
  var set;
  if (add.arguments.length == 2)
   {
    set = add.arguments[1];
   }
  else if (add.arguments.length == 1)
   {
    set = this.plainMC;
   }
  else
   {
    throw "add(): " + add.arguments.length + " arguments given; expected 1 or 2";
   }
  var rhs = add.arguments[0];
  //--com.ibm.icu.math.BigDecimal lhs;
  var lhs;
  //--int reqdig;
  var reqdig;
  //--com.ibm.icu.math.BigDecimal res;
  var res;
  //--byte usel[];
  var usel;
  //--int usellen;
  var usellen;
  //--byte user[];
  var user;
  //--int userlen;
  var userlen;
  //--int newlen=0;
  var newlen=0;
  //--int tlen=0;
  var tlen=0;
  //--int mult=0;
  var mult=0;
  //--byte t[]=null;
  var t=null;
  //--int ia=0;
  var ia=0;
  //--int ib=0;
  var ib=0;
  //--int ea=0;
  var ea=0;
  //int eb=0;
  var eb=0;
  //byte ca=0;
  var ca=0;
  //--byte cb=0;
  var cb=0;
  /* determine requested digits and form */
  if (set.lostDigits)
   this.checkdigits(rhs,set.digits);
  lhs=this; // name for clarity and proxy

  /* Quick exit for add floating 0 */
  // plus() will optimize to return same object if possible
  if (lhs.ind==0)
   if (set.form!=MathContext.prototype.PLAIN)
    return rhs.plus(set);
  if (rhs.ind==0)
   if (set.form!=MathContext.prototype.PLAIN)
    return lhs.plus(set);

  /* Prepare numbers (round, unless unlimited precision) */
  reqdig=set.digits; // local copy (heavily used)
  if (reqdig>0)
   {
    if (lhs.mant.length>reqdig)
     lhs=this.clone(lhs).round(set);
    if (rhs.mant.length>reqdig)
     rhs=this.clone(rhs).round(set);
   // [we could reuse the new LHS for result in this case]
   }

  res=new BigDecimal(); // build result here

  /* Now see how much we have to pad or truncate lhs or rhs in order
     to align the numbers.  If one number is much larger than the
     other, then the smaller cannot affect the answer [but we may
     still need to pad with up to DIGITS trailing zeros]. */
  // Note sign may be 0 if digits (reqdig) is 0
  // usel and user will be the byte arrays passed to the adder; we'll
  // use them on all paths except quick exits
  usel=lhs.mant;
  usellen=lhs.mant.length;
  user=rhs.mant;
  userlen=rhs.mant.length;
  {padder:do{/*select*/
  if (lhs.exp==rhs.exp)
   {/* no padding needed */
    // This is the most common, and fastest, path
    res.exp=lhs.exp;
   }
  else if (lhs.exp>rhs.exp)
   { // need to pad lhs and/or truncate rhs
    newlen=(usellen+lhs.exp)-rhs.exp;
    /* If, after pad, lhs would be longer than rhs by digits+1 or
       more (and digits>0) then rhs cannot affect answer, so we only
       need to pad up to a length of DIGITS+1. */
    if (newlen>=((userlen+reqdig)+1))
     if (reqdig>0)
      {
       // LHS is sufficient
       res.mant=usel;
       res.exp=lhs.exp;
       res.ind=lhs.ind;
       if (usellen<reqdig)
        { // need 0 padding
         res.mant=this.extend(lhs.mant,reqdig);
         res.exp=res.exp-((reqdig-usellen));
        }
       return res.finish(set,false);
      }
    // RHS may affect result
    res.exp=rhs.exp; // expected final exponent
    if (newlen>(reqdig+1))
     if (reqdig>0)
      {
       // LHS will be max; RHS truncated
       tlen=(newlen-reqdig)-1; // truncation length
       userlen=userlen-tlen;
       res.exp=res.exp+tlen;
       newlen=reqdig+1;
      }
    if (newlen>usellen)
     usellen=newlen; // need to pad LHS
   }
  else{ // need to pad rhs and/or truncate lhs
   newlen=(userlen+rhs.exp)-lhs.exp;
   if (newlen>=((usellen+reqdig)+1))
    if (reqdig>0)
     {
      // RHS is sufficient
      res.mant=user;
      res.exp=rhs.exp;
      res.ind=rhs.ind;
      if (userlen<reqdig)
       { // need 0 padding
        res.mant=this.extend(rhs.mant,reqdig);
        res.exp=res.exp-((reqdig-userlen));
       }
      return res.finish(set,false);
     }
   // LHS may affect result
   res.exp=lhs.exp; // expected final exponent
   if (newlen>(reqdig+1))
    if (reqdig>0)
     {
      // RHS will be max; LHS truncated
      tlen=(newlen-reqdig)-1; // truncation length
      usellen=usellen-tlen;
      res.exp=res.exp+tlen;
      newlen=reqdig+1;
     }
   if (newlen>userlen)
    userlen=newlen; // need to pad RHS
  }
  }while(false);}/*padder*/

  /* OK, we have aligned mantissas.  Now add or subtract. */
  // 1998.06.27 Sign may now be 0 [e.g., 0.000] .. treat as positive
  // 1999.05.27 Allow for 00 on lhs [is not larger than 2 on rhs]
  // 1999.07.10 Allow for 00 on rhs [is not larger than 2 on rhs]
  if (lhs.ind==this.iszero)
   res.ind=this.ispos;
  else
   res.ind=lhs.ind; // likely sign, all paths
  if (((lhs.ind==this.isneg)?1:0)==((rhs.ind==this.isneg)?1:0))  // same sign, 0 non-negative
   mult=1;
  else
   {signdiff:do{ // different signs, so subtraction is needed
    mult=-1; // will cause subtract
    /* Before we can subtract we must determine which is the larger,
       as our add/subtract routine only handles non-negative results
       so we may need to swap the operands. */
    {swaptest:do{/*select*/
    if (rhs.ind==this.iszero)
     {} // original A bigger
    else if ((usellen<userlen)||(lhs.ind==this.iszero))
     { // original B bigger
      t=usel;
      usel=user;
      user=t; // swap
      tlen=usellen;
      usellen=userlen;
      userlen=tlen; // ..
      res.ind=-res.ind; // and set sign
     }
    else if (usellen>userlen)
     {} // original A bigger
    else{
     {/* logical lengths the same */ // need compare
      /* may still need to swap: compare the strings */
      ia=0;
      ib=0;
      ea=usel.length-1;
      eb=user.length-1;
      {compare:for(;;){
       if (ia<=ea)
        ca=usel[ia];
       else
        {
         if (ib>eb)
          {/* identical */
           if (set.form!=MathContext.prototype.PLAIN)
            return this.ZERO;
           // [if PLAIN we must do the subtract, in case of 0.000 results]
           break compare;
          }
         ca=0;
        }
       if (ib<=eb)
        cb=user[ib];
       else
        cb=0;
       if (ca!=cb)
        {
         if (ca<cb)
          {/* swap needed */
           t=usel;
           usel=user;
           user=t; // swap
           tlen=usellen;
           usellen=userlen;
           userlen=tlen; // ..
           res.ind=-res.ind;
          }
         break compare;
        }
       /* mantissas the same, so far */
       ia++;
       ib++;
       }
      }/*compare*/
     } // lengths the same
    }
    }while(false);}/*swaptest*/
   }while(false);}/*signdiff*/

  /* here, A is > B if subtracting */
  // add [A+B*1] or subtract [A+(B*-1)]
  res.mant=this.byteaddsub(usel,usellen,user,userlen,mult,false);
  // [reuse possible only after chop; accounting makes not worthwhile]

  // Finish() rounds before stripping leading 0's, then sets form, etc.
  return res.finish(set,false);
  }

 /**
  * Compares this <code>BigDecimal</code> to another, using unlimited
  * precision.
  * <p>
  * The same as {@link #compareTo(BigDecimal, MathContext)},
  * where the <code>BigDecimal</code> is <code>rhs</code>,
  * and the context is <code>new MathContext(0, MathContext.PLAIN)</code>.
  *
  * @param  rhs The <code>BigDecimal</code> for the right hand side of
  *             the comparison.
  * @return     An <code>int</code> whose value is -1, 0, or 1 as
  *             <code>this</code> is numerically less than, equal to,
  *             or greater than <code>rhs</code>.
  * @see    #compareTo(Object)
  * @stable ICU 2.0
  */

 //--public int compareTo(com.ibm.icu.math.BigDecimal rhs){
 //-- return this.compareTo(rhs,plainMC);
 //-- }

 /**
  * Compares this <code>BigDecimal</code> to another.
  * <p>
  * Implements numeric comparison,
  * (as defined in the decimal documentation, see {@link BigDecimal
  * class header}),
  * and returns a result of type <code>int</code>.
  * <p>
  * The result will be:
  * <table cellpadding=2><tr>
  * <td align=right><b>-1</b></td>
  * <td>if the current object is less than the first parameter</td>
  * </tr><tr>
  * <td align=right><b>0</b></td>
  * <td>if the current object is equal to the first parameter</td>
  * </tr><tr>
  * <td align=right><b>1</b></td>
  * <td>if the current object is greater than the first parameter.</td>
  * </tr></table>
  * <p>
  * A {@link #compareTo(Object)} method is also provided.
  *
  * @param  rhs The <code>BigDecimal</code> for the right hand side of
  *             the comparison.
  * @param  set The <code>MathContext</code> arithmetic settings.
  * @return     An <code>int</code> whose value is -1, 0, or 1 as
  *             <code>this</code> is numerically less than, equal to,
  *             or greater than <code>rhs</code>.
  * @see    #compareTo(Object)
  * @stable ICU 2.0
  */

 //public int compareTo(com.ibm.icu.math.BigDecimal rhs,com.ibm.icu.math.MathContext set){
 function compareTo() {
  var set;
  if (compareTo.arguments.length == 2)
   {
    set = compareTo.arguments[1];
   }
  else if (compareTo.arguments.length == 1)
   {
    set = this.plainMC;
   }
  else
   {
    throw "compareTo(): " + compareTo.arguments.length + " arguments given; expected 1 or 2";
   }
  var rhs = compareTo.arguments[0];
  //--int thislength=0;
  var thislength=0;
  //--int i=0;
  var i=0;
  //--com.ibm.icu.math.BigDecimal newrhs;
  var newrhs;
  // rhs=null will raise NullPointerException, as per Comparable interface
  if (set.lostDigits)
   this.checkdigits(rhs,set.digits);
  // [add will recheck in slowpath cases .. but would report -rhs]
  if ((this.ind==rhs.ind)&&(this.exp==rhs.exp))
   {
    /* sign & exponent the same [very common] */
    thislength=this.mant.length;
    if (thislength<rhs.mant.length)
     return -this.ind;
    if (thislength>rhs.mant.length)
     return this.ind;
    /* lengths are the same; we can do a straight mantissa compare
       unless maybe rounding [rounding is very unusual] */
    if ((thislength<=set.digits)||(set.digits==0))
     {
      {var $6=thislength;i=0;i:for(;$6>0;$6--,i++){
       if (this.mant[i]<rhs.mant[i])
        return -this.ind;
       if (this.mant[i]>rhs.mant[i])
        return this.ind;
       }
      }/*i*/
      return 0; // identical
     }
   /* drop through for full comparison */
   }
  else
   {
    /* More fastpaths possible */
    if (this.ind<rhs.ind)
     return -1;
    if (this.ind>rhs.ind)
     return 1;
   }
  /* carry out a subtract to make the comparison */
  newrhs=this.clone(rhs); // safe copy
  newrhs.ind=-newrhs.ind; // prepare to subtract
  return this.add(newrhs,set).ind; // add, and return sign of result
  }

 /**
  * Returns a plain <code>BigDecimal</code> whose value is
  * <code>this/rhs</code>, using fixed point arithmetic.
  * <p>
  * The same as {@link #divide(BigDecimal, int)},
  * where the <code>BigDecimal</code> is <code>rhs</code>,
  * and the rounding mode is {@link MathContext#ROUND_HALF_UP}.
  *
  * The length of the decimal part (the scale) of the result will be
  * the same as the scale of the current object, if the latter were
  * formatted without exponential notation.
  *
  * @param  rhs The <code>BigDecimal</code> for the right hand side of
  *             the division.
  * @return     A plain <code>BigDecimal</code> whose value is
  *             <code>this/rhs</code>, using fixed point arithmetic.
  * @throws ArithmeticException if <code>rhs</code> is zero.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal divide(com.ibm.icu.math.BigDecimal rhs){
 //-- return this.dodivide('D',rhs,plainMC,-1);
 //-- }

 /**
  * Returns a plain <code>BigDecimal</code> whose value is
  * <code>this/rhs</code>, using fixed point arithmetic and a
  * rounding mode.
  * <p>
  * The same as {@link #divide(BigDecimal, int, int)},
  * where the <code>BigDecimal</code> is <code>rhs</code>,
  * and the second parameter is <code>this.scale()</code>, and
  * the third is <code>round</code>.
  * <p>
  * The length of the decimal part (the scale) of the result will
  * therefore be the same as the scale of the current object, if the
  * latter were formatted without exponential notation.
  * <p>
  * @param  rhs   The <code>BigDecimal</code> for the right hand side of
  *               the division.
  * @param  round The <code>int</code> rounding mode to be used for
  *               the division (see the {@link MathContext} class).
  * @return       A plain <code>BigDecimal</code> whose value is
  *               <code>this/rhs</code>, using fixed point arithmetic
  *               and the specified rounding mode.
  * @throws IllegalArgumentException if <code>round</code> is not a
  *               valid rounding mode.
  * @throws ArithmeticException if <code>rhs</code> is zero.
  * @throws ArithmeticException if <code>round</code> is {@link
  *               MathContext#ROUND_UNNECESSARY} and
  *               <code>this.scale()</code> is insufficient to
  *               represent the result exactly.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal divide(com.ibm.icu.math.BigDecimal rhs,int round){
 //-- com.ibm.icu.math.MathContext set;
 //-- set=new com.ibm.icu.math.MathContext(0,com.ibm.icu.math.MathContext.PLAIN,false,round); // [checks round, too]
 //-- return this.dodivide('D',rhs,set,-1); // take scale from LHS
 //-- }

 /**
  * Returns a plain <code>BigDecimal</code> whose value is
  * <code>this/rhs</code>, using fixed point arithmetic and a
  * given scale and rounding mode.
  * <p>
  * The same as {@link #divide(BigDecimal, MathContext)},
  * where the <code>BigDecimal</code> is <code>rhs</code>,
  * <code>new MathContext(0, MathContext.PLAIN, false, round)</code>,
  * except that the length of the decimal part (the scale) to be used
  * for the result is explicit rather than being taken from
  * <code>this</code>.
  * <p>
  * The length of the decimal part (the scale) of the result will be
  * the same as the scale of the current object, if the latter were
  * formatted without exponential notation.
  * <p>
  * @param  rhs   The <code>BigDecimal</code> for the right hand side of
  *               the division.
  * @param  scale The <code>int</code> scale to be used for the result.
  * @param  round The <code>int</code> rounding mode to be used for
  *               the division (see the {@link MathContext} class).
  * @return       A plain <code>BigDecimal</code> whose value is
  *               <code>this/rhs</code>, using fixed point arithmetic
  *               and the specified rounding mode.
  * @throws IllegalArgumentException if <code>round</code> is not a
  *               valid rounding mode.
  * @throws ArithmeticException if <code>rhs</code> is zero.
  * @throws ArithmeticException if <code>scale</code> is negative.
  * @throws ArithmeticException if <code>round</code> is {@link
  *               MathContext#ROUND_UNNECESSARY} and <code>scale</code>
  *               is insufficient to represent the result exactly.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal divide(com.ibm.icu.math.BigDecimal rhs,int scale,int round){
 //-- com.ibm.icu.math.MathContext set;
 //-- if (scale<0)
 //--  throw new java.lang.ArithmeticException("Negative scale:"+" "+scale);
 //-- set=new com.ibm.icu.math.MathContext(0,com.ibm.icu.math.MathContext.PLAIN,false,round); // [checks round]
 //-- return this.dodivide('D',rhs,set,scale);
 //-- }

 /**
  * Returns a <code>BigDecimal</code> whose value is <code>this/rhs</code>.
  * <p>
  * Implements the division (<b><code>/</code></b>) operator
  * (as defined in the decimal documentation, see {@link BigDecimal
  * class header}),
  * and returns the result as a <code>BigDecimal</code> object.
  *
  * @param  rhs The <code>BigDecimal</code> for the right hand side of
  *             the division.
  * @param  set The <code>MathContext</code> arithmetic settings.
  * @return     A <code>BigDecimal</code> whose value is
  *             <code>this/rhs</code>.
  * @throws ArithmeticException if <code>rhs</code> is zero.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal divide(com.ibm.icu.math.BigDecimal rhs,com.ibm.icu.math.MathContext set){
 function divide() {
  var set;
  var scale = -1;
  if (divide.arguments.length == 2)
   {
    if (typeof divide.arguments[1] == 'number')
     {
      set=new MathContext(0,MathContext.prototype.PLAIN,false,divide.arguments[1]); // [checks round, too]
     }
    else
     {
      set = divide.arguments[1];
     }
   }
  else if (divide.arguments.length == 3)
   {
    scale = divide.arguments[1];
    if (scale<0)
     throw "divide(): Negative scale: "+scale;
    set=new MathContext(0,MathContext.prototype.PLAIN,false,divide.arguments[2]); // [checks round]
   }
  else if (divide.arguments.length == 1)
   {
    set = this.plainMC;
   }
  else
   {
    throw "divide(): " + divide.arguments.length + " arguments given; expected between 1 and 3";
   }
  var rhs = divide.arguments[0];
  return this.dodivide('D',rhs,set,scale);
  }

 /**
  * Returns a plain <code>BigDecimal</code> whose value is the integer
  * part of <code>this/rhs</code>.
  * <p>
  * The same as {@link #divideInteger(BigDecimal, MathContext)},
  * where the <code>BigDecimal</code> is <code>rhs</code>,
  * and the context is <code>new MathContext(0, MathContext.PLAIN)</code>.
  *
  * @param  rhs The <code>BigDecimal</code> for the right hand side of
  *             the integer division.
  * @return     A <code>BigDecimal</code> whose value is the integer
  *             part of <code>this/rhs</code>.
  * @throws ArithmeticException if <code>rhs</code> is zero.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal divideInteger(com.ibm.icu.math.BigDecimal rhs){
 //-- // scale 0 to drop .000 when plain
 //-- return this.dodivide('I',rhs,plainMC,0);
 //-- }

 /**
  * Returns a <code>BigDecimal</code> whose value is the integer
  * part of <code>this/rhs</code>.
  * <p>
  * Implements the integer division operator
  * (as defined in the decimal documentation, see {@link BigDecimal
  * class header}),
  * and returns the result as a <code>BigDecimal</code> object.
  *
  * @param  rhs The <code>BigDecimal</code> for the right hand side of
  *             the integer division.
  * @param  set The <code>MathContext</code> arithmetic settings.
  * @return     A <code>BigDecimal</code> whose value is the integer
  *             part of <code>this/rhs</code>.
  * @throws ArithmeticException if <code>rhs</code> is zero.
  * @throws ArithmeticException if the result will not fit in the
  *             number of digits specified for the context.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal divideInteger(com.ibm.icu.math.BigDecimal rhs,com.ibm.icu.math.MathContext set){
 function divideInteger() {
  var set;
  if (divideInteger.arguments.length == 2)
   {
    set = divideInteger.arguments[1];
   }
  else if (divideInteger.arguments.length == 1)
   {
    set = this.plainMC;
   }
  else
   {
    throw "divideInteger(): " + divideInteger.arguments.length + " arguments given; expected 1 or 2";
   }
  var rhs = divideInteger.arguments[0];
  // scale 0 to drop .000 when plain
  return this.dodivide('I',rhs,set,0);
  }

 /**
  * Returns a plain <code>BigDecimal</code> whose value is
  * the maximum of <code>this</code> and <code>rhs</code>.
  * <p>
  * The same as {@link #max(BigDecimal, MathContext)},
  * where the <code>BigDecimal</code> is <code>rhs</code>,
  * and the context is <code>new MathContext(0, MathContext.PLAIN)</code>.
  *
  * @param  rhs The <code>BigDecimal</code> for the right hand side of
  *             the comparison.
  * @return     A <code>BigDecimal</code> whose value is
  *             the maximum of <code>this</code> and <code>rhs</code>.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal max(com.ibm.icu.math.BigDecimal rhs){
 //-- return this.max(rhs,plainMC);
 //-- }

 /**
  * Returns a <code>BigDecimal</code> whose value is
  * the maximum of <code>this</code> and <code>rhs</code>.
  * <p>
  * Returns the larger of the current object and the first parameter.
  * <p>
  * If calling the {@link #compareTo(BigDecimal, MathContext)} method
  * with the same parameters would return <code>1</code> or
  * <code>0</code>, then the result of calling the
  * {@link #plus(MathContext)} method on the current object (using the
  * same <code>MathContext</code> parameter) is returned.
  * Otherwise, the result of calling the {@link #plus(MathContext)}
  * method on the first parameter object (using the same
  * <code>MathContext</code> parameter) is returned.
  *
  * @param  rhs The <code>BigDecimal</code> for the right hand side of
  *             the comparison.
  * @param  set The <code>MathContext</code> arithmetic settings.
  * @return     A <code>BigDecimal</code> whose value is
  *             the maximum of <code>this</code> and <code>rhs</code>.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal max(com.ibm.icu.math.BigDecimal rhs,com.ibm.icu.math.MathContext set){
 function max() {
  var set;
  if (max.arguments.length == 2)
   {
    set = max.arguments[1];
   }
  else if (max.arguments.length == 1)
   {
    set = this.plainMC;
   }
  else
   {
    throw "max(): " + max.arguments.length + " arguments given; expected 1 or 2";
   }
  var rhs = max.arguments[0];
  if ((this.compareTo(rhs,set))>=0)
   return this.plus(set);
  else
   return rhs.plus(set);
  }

 /**
  * Returns a plain <code>BigDecimal</code> whose value is
  * the minimum of <code>this</code> and <code>rhs</code>.
  * <p>
  * The same as {@link #min(BigDecimal, MathContext)},
  * where the <code>BigDecimal</code> is <code>rhs</code>,
  * and the context is <code>new MathContext(0, MathContext.PLAIN)</code>.
  *
  * @param  rhs The <code>BigDecimal</code> for the right hand side of
  *             the comparison.
  * @return     A <code>BigDecimal</code> whose value is
  *             the minimum of <code>this</code> and <code>rhs</code>.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal min(com.ibm.icu.math.BigDecimal rhs){
 //-- return this.min(rhs,plainMC);
 //-- }

 /**
  * Returns a <code>BigDecimal</code> whose value is
  * the minimum of <code>this</code> and <code>rhs</code>.
  * <p>
  * Returns the smaller of the current object and the first parameter.
  * <p>
  * If calling the {@link #compareTo(BigDecimal, MathContext)} method
  * with the same parameters would return <code>-1</code> or
  * <code>0</code>, then the result of calling the
  * {@link #plus(MathContext)} method on the current object (using the
  * same <code>MathContext</code> parameter) is returned.
  * Otherwise, the result of calling the {@link #plus(MathContext)}
  * method on the first parameter object (using the same
  * <code>MathContext</code> parameter) is returned.
  *
  * @param  rhs The <code>BigDecimal</code> for the right hand side of
  *             the comparison.
  * @param  set The <code>MathContext</code> arithmetic settings.
  * @return     A <code>BigDecimal</code> whose value is
  *             the minimum of <code>this</code> and <code>rhs</code>.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal min(com.ibm.icu.math.BigDecimal rhs,com.ibm.icu.math.MathContext set){
 function min() {
  var set;
  if (min.arguments.length == 2)
   {
    set = min.arguments[1];
   }
  else if (min.arguments.length == 1)
   {
    set = this.plainMC;
   }
  else
   {
    throw "min(): " + min.arguments.length + " arguments given; expected 1 or 2";
   }
  var rhs = min.arguments[0];
  if ((this.compareTo(rhs,set))<=0)
   return this.plus(set);
  else
   return rhs.plus(set);
  }

 /**
  * Returns a plain <code>BigDecimal</code> whose value is
  * <code>this*rhs</code>, using fixed point arithmetic.
  * <p>
  * The same as {@link #add(BigDecimal, MathContext)},
  * where the <code>BigDecimal</code> is <code>rhs</code>,
  * and the context is <code>new MathContext(0, MathContext.PLAIN)</code>.
  * <p>
  * The length of the decimal part (the scale) of the result will be
  * the sum of the scales of the operands, if they were formatted
  * without exponential notation.
  *
  * @param  rhs The <code>BigDecimal</code> for the right hand side of
  *             the multiplication.
  * @return     A <code>BigDecimal</code> whose value is
  *             <code>this*rhs</code>, using fixed point arithmetic.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal multiply(com.ibm.icu.math.BigDecimal rhs){
 //-- return this.multiply(rhs,plainMC);
 //-- }

 /**
  * Returns a <code>BigDecimal</code> whose value is <code>this*rhs</code>.
  * <p>
  * Implements the multiplication (<b><code>*</code></b>) operator
  * (as defined in the decimal documentation, see {@link BigDecimal
  * class header}),
  * and returns the result as a <code>BigDecimal</code> object.
  *
  * @param  rhs The <code>BigDecimal</code> for the right hand side of
  *             the multiplication.
  * @param  set The <code>MathContext</code> arithmetic settings.
  * @return     A <code>BigDecimal</code> whose value is
  *             <code>this*rhs</code>.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal multiply(com.ibm.icu.math.BigDecimal rhs,com.ibm.icu.math.MathContext set){
 function multiply() {
  var set;
  if (multiply.arguments.length == 2)
   {
    set = multiply.arguments[1];
   }
  else if (multiply.arguments.length == 1)
   {
    set = this.plainMC;
   }
  else
   {
    throw "multiply(): " + multiply.arguments.length + " arguments given; expected 1 or 2";
   }
  var rhs = multiply.arguments[0];
  //--com.ibm.icu.math.BigDecimal lhs;
  var lhs;
  //--int padding;
  var padding;
  //--int reqdig;
  var reqdig;
  //--byte multer[]=null;
  var multer=null;
  //--byte multand[]=null;
  var multand=null;
  //--int multandlen;
  var multandlen;
  //--int acclen=0;
  var acclen=0;
  //--com.ibm.icu.math.BigDecimal res;
  var res;
  //--byte acc[];
  var acc;
  //--int n=0;
  var n=0;
  //--byte mult=0;
  var mult=0;
  if (set.lostDigits)
   this.checkdigits(rhs,set.digits);
  lhs=this; // name for clarity and proxy

  /* Prepare numbers (truncate, unless unlimited precision) */
  padding=0; // trailing 0's to add
  reqdig=set.digits; // local copy
  if (reqdig>0)
   {
    if (lhs.mant.length>reqdig)
     lhs=this.clone(lhs).round(set);
    if (rhs.mant.length>reqdig)
     rhs=this.clone(rhs).round(set);
   // [we could reuse the new LHS for result in this case]
   }
  else
   {/* unlimited */
    // fixed point arithmetic will want every trailing 0; we add these
    // after the calculation rather than before, for speed.
    if (lhs.exp>0)
     padding=padding+lhs.exp;
    if (rhs.exp>0)
     padding=padding+rhs.exp;
   }

  // For best speed, as in DMSRCN, we use the shorter number as the
  // multiplier and the longer as the multiplicand.
  // 1999.12.22: We used to special case when the result would fit in
  //             a long, but with Java 1.3 this gave no advantage.
  if (lhs.mant.length<rhs.mant.length)
   {
    multer=lhs.mant;
    multand=rhs.mant;
   }
  else
   {
    multer=rhs.mant;
    multand=lhs.mant;
   }

  /* Calculate how long result byte array will be */
  multandlen=(multer.length+multand.length)-1; // effective length
  // optimize for 75% of the cases where a carry is expected...
  if ((multer[0]*multand[0])>9)
   acclen=multandlen+1;
  else
   acclen=multandlen;

  /* Now the main long multiplication loop */
  res=new BigDecimal(); // where we'll build result
  acc=this.createArrayWithZeros(acclen); // accumulator, all zeros
  // 1998.07.01: calculate from left to right so that accumulator goes
  // to likely final length on first addition; this avoids a one-digit
  // extension (and object allocation) each time around the loop.
  // Initial number therefore has virtual zeros added to right.
  {var $7=multer.length;n=0;n:for(;$7>0;$7--,n++){
   mult=multer[n];
   if (mult!=0)
    { // [optimization]
     // accumulate [accumulator is reusable array]
     acc=this.byteaddsub(acc,acc.length,multand,multandlen,mult,true);
    }
   // divide multiplicand by 10 for next digit to right
   multandlen--; // 'virtual length'
   }
  }/*n*/

  res.ind=lhs.ind*rhs.ind; // final sign
  res.exp=(lhs.exp+rhs.exp)-padding; // final exponent
  // [overflow is checked by finish]

  /* add trailing zeros to the result, if necessary */
  if (padding==0)
   res.mant=acc;
  else
   res.mant=this.extend(acc,acc.length+padding); // add trailing 0s
  return res.finish(set,false);
  }

 /**
  * Returns a plain <code>BigDecimal</code> whose value is
  * <code>-this</code>.
  * <p>
  * The same as {@link #negate(MathContext)}, where the context is
  * <code>new MathContext(0, MathContext.PLAIN)</code>.
  * <p>
  * The length of the decimal part (the scale) of the result will be
  * be <code>this.scale()</code>
  *
  *
  * @return A <code>BigDecimal</code> whose value is
  *         <code>-this</code>.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal negate(){
 //-- return this.negate(plainMC);
 //-- }

 /**
  * Returns a <code>BigDecimal</code> whose value is <code>-this</code>.
  * <p>
  * Implements the negation (Prefix <b><code>-</code></b>) operator
  * (as defined in the decimal documentation, see {@link BigDecimal
  * class header}),
  * and returns the result as a <code>BigDecimal</code> object.
  *
  * @param  set The <code>MathContext</code> arithmetic settings.
  * @return A <code>BigDecimal</code> whose value is
  *         <code>-this</code>.
  * @stable ICU 2.0
  */

 //public com.ibm.icu.math.BigDecimal negate(com.ibm.icu.math.MathContext set){
 function negate() {
  var set;
  if (negate.arguments.length == 1)
   {
    set = negate.arguments[0];
   }
  else if (negate.arguments.length == 0)
   {
    set = this.plainMC;
   }
  else
   {
    throw "negate(): " + negate.arguments.length + " arguments given; expected 0 or 1";
   }
  //--com.ibm.icu.math.BigDecimal res;
  var res;
  // Originally called minus(), changed to matched Java precedents
  // This simply clones, flips the sign, and possibly rounds
  if (set.lostDigits)
   this.checkdigits(null,set.digits);
  res=this.clone(this); // safe copy
  res.ind=-res.ind;
  return res.finish(set,false);
  }

 /**
  * Returns a plain <code>BigDecimal</code> whose value is
  * <code>+this</code>.
  * Note that <code>this</code> is not necessarily a
  * plain <code>BigDecimal</code>, but the result will always be.
  * <p>
  * The same as {@link #plus(MathContext)}, where the context is
  * <code>new MathContext(0, MathContext.PLAIN)</code>.
  * <p>
  * The length of the decimal part (the scale) of the result will be
  * be <code>this.scale()</code>
  *
  * @return A <code>BigDecimal</code> whose value is
  *         <code>+this</code>.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal plus(){
 //-- return this.plus(plainMC);
 //-- }

 /**
  * Returns a <code>BigDecimal</code> whose value is
  * <code>+this</code>.
  * <p>
  * Implements the plus (Prefix <b><code>+</code></b>) operator
  * (as defined in the decimal documentation, see {@link BigDecimal
  * class header}),
  * and returns the result as a <code>BigDecimal</code> object.
  * <p>
  * This method is useful for rounding or otherwise applying a context
  * to a decimal value.
  *
  * @param  set The <code>MathContext</code> arithmetic settings.
  * @return A <code>BigDecimal</code> whose value is
  *         <code>+this</code>.
  * @stable ICU 2.0
  */

 //public com.ibm.icu.math.BigDecimal plus(com.ibm.icu.math.MathContext set){
 function plus() {
  var set;
  if (plus.arguments.length == 1)
   {
    set = plus.arguments[0];
   }
  else if (plus.arguments.length == 0)
   {
    set = this.plainMC;
   }
  else
   {
    throw "plus(): " + plus.arguments.length + " arguments given; expected 0 or 1";
   }
  // This clones and forces the result to the new settings
  // May return same object
  if (set.lostDigits)
   this.checkdigits(null,set.digits);
  // Optimization: returns same object for some common cases
  if (set.form==MathContext.prototype.PLAIN)
   if (this.form==MathContext.prototype.PLAIN)
    {
     if (this.mant.length<=set.digits)
      return this;
     if (set.digits==0)
      return this;
    }
  return this.clone(this).finish(set,false);
  }

 /**
  * Returns a plain <code>BigDecimal</code> whose value is
  * <code>this**rhs</code>, using fixed point arithmetic.
  * <p>
  * The same as {@link #pow(BigDecimal, MathContext)},
  * where the <code>BigDecimal</code> is <code>rhs</code>,
  * and the context is <code>new MathContext(0, MathContext.PLAIN)</code>.
  * <p>
  * The parameter is the power to which the <code>this</code> will be
  * raised; it must be in the range 0 through 999999999, and must
  * have a decimal part of zero.  Note that these restrictions may be
  * removed in the future, so they should not be used as a test for a
  * whole number.
  * <p>
  * In addition, the power must not be negative, as no
  * <code>MathContext</code> is used and so the result would then
  * always be 0.
  *
  * @param  rhs The <code>BigDecimal</code> for the right hand side of
  *             the operation (the power).
  * @return     A <code>BigDecimal</code> whose value is
  *             <code>this**rhs</code>, using fixed point arithmetic.
  * @throws ArithmeticException if <code>rhs</code> is out of range or
  *             is not a whole number.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal pow(com.ibm.icu.math.BigDecimal rhs){
 //-- return this.pow(rhs,plainMC);
 //-- }
 // The name for this method is inherited from the precedent set by the
 // BigInteger and Math classes.

 /**
  * Returns a <code>BigDecimal</code> whose value is <code>this**rhs</code>.
  * <p>
  * Implements the power (<b><code>**</code></b>) operator
  * (as defined in the decimal documentation, see {@link BigDecimal
  * class header}),
  * and returns the result as a <code>BigDecimal</code> object.
  * <p>
  * The first parameter is the power to which the <code>this</code>
  * will be raised; it must be in the range -999999999 through
  * 999999999, and must have a decimal part of zero.  Note that these
  * restrictions may be removed in the future, so they should not be
  * used as a test for a whole number.
  * <p>
  * If the <code>digits</code> setting of the <code>MathContext</code>
  * parameter is 0, the power must be zero or positive.
  *
  * @param  rhs The <code>BigDecimal</code> for the right hand side of
  *             the operation (the power).
  * @param  set The <code>MathContext</code> arithmetic settings.
  * @return     A <code>BigDecimal</code> whose value is
  *             <code>this**rhs</code>.
  * @throws ArithmeticException if <code>rhs</code> is out of range or
  *             is not a whole number.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal pow(com.ibm.icu.math.BigDecimal rhs,com.ibm.icu.math.MathContext set){
 function pow() {
  var set;
  if (pow.arguments.length == 2)
   {
    set = pow.arguments[1];
   }
  else if (pow.arguments.length == 1)
   {
    set = this.plainMC;
   }
  else
   {
    throw "pow(): " + pow.arguments.length + " arguments given; expected 1 or 2";
   }
  var rhs = pow.arguments[0];
  //--int n;
  var n;
  //--com.ibm.icu.math.BigDecimal lhs;
  var lhs;
  //--int reqdig;
  var reqdig;
  //-- int workdigits=0;
  var workdigits=0;
  //--int L=0;
  var L=0;
  //--com.ibm.icu.math.MathContext workset;
  var workset;
  //--com.ibm.icu.math.BigDecimal res;
  var res;
  //--boolean seenbit;
  var seenbit;
  //--int i=0;
  var i=0;
  if (set.lostDigits)
   this.checkdigits(rhs,set.digits);
  n=rhs.intcheck(this.MinArg,this.MaxArg); // check RHS by the rules
  lhs=this; // clarified name

  reqdig=set.digits; // local copy (heavily used)
  if (reqdig==0)
   {
    if (rhs.ind==this.isneg)
     //--throw new java.lang.ArithmeticException("Negative power:"+" "+rhs.toString());
     throw "pow(): Negative power: " + rhs.toString();
    workdigits=0;
   }
  else
   {/* non-0 digits */
    if ((rhs.mant.length+rhs.exp)>reqdig)
     //--throw new java.lang.ArithmeticException("Too many digits:"+" "+rhs.toString());
     throw "pow(): Too many digits: " + rhs.toString();

    /* Round the lhs to DIGITS if need be */
    if (lhs.mant.length>reqdig)
     lhs=this.clone(lhs).round(set);

    /* L for precision calculation [see ANSI X3.274-1996] */
    L=rhs.mant.length+rhs.exp; // length without decimal zeros/exp
    workdigits=(reqdig+L)+1; // calculate the working DIGITS
   }

  /* Create a copy of set for working settings */
  // Note: no need to check for lostDigits again.
  // 1999.07.17 Note: this construction must follow RHS check
  workset=new MathContext(workdigits,set.form,false,set.roundingMode);

  res=this.ONE; // accumulator
  if (n==0)
   return res; // x**0 == 1
  if (n<0)
   n=-n; // [rhs.ind records the sign]
  seenbit=false; // set once we've seen a 1-bit
  {i=1;i:for(;;i++){ // for each bit [top bit ignored]
   //n=n+n; // shift left 1 bit
   n<<=1;
   if (n<0)
    { // top bit is set
     seenbit=true; // OK, we're off
     res=res.multiply(lhs,workset); // acc=acc*x
    }
   if (i==31)
    break i; // that was the last bit
   if ((!seenbit))
    continue i; // we don't have to square 1
   res=res.multiply(res,workset); // acc=acc*acc [square]
   }
  }/*i*/ // 32 bits
  if (rhs.ind<0)  // was a **-n [hence digits>0]
   res=this.ONE.divide(res,workset); // .. so acc=1/acc
  return res.finish(set,true); // round and strip [original digits]
  }

 /**
  * Returns a plain <code>BigDecimal</code> whose value is
  * the remainder of <code>this/rhs</code>, using fixed point arithmetic.
  * <p>
  * The same as {@link #remainder(BigDecimal, MathContext)},
  * where the <code>BigDecimal</code> is <code>rhs</code>,
  * and the context is <code>new MathContext(0, MathContext.PLAIN)</code>.
  * <p>
  * This is not the modulo operator -- the result may be negative.
  *
  * @param  rhs The <code>BigDecimal</code> for the right hand side of
  *             the remainder operation.
  * @return     A <code>BigDecimal</code> whose value is the remainder
  *             of <code>this/rhs</code>, using fixed point arithmetic.
  * @throws ArithmeticException if <code>rhs</code> is zero.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal remainder(com.ibm.icu.math.BigDecimal rhs){
 //-- return this.dodivide('R',rhs,plainMC,-1);
 //-- }

 /**
  * Returns a <code>BigDecimal</code> whose value is the remainder of
  * <code>this/rhs</code>.
  * <p>
  * Implements the remainder operator
  * (as defined in the decimal documentation, see {@link BigDecimal
  * class header}),
  * and returns the result as a <code>BigDecimal</code> object.
  * <p>
  * This is not the modulo operator -- the result may be negative.
  *
  * @param  rhs The <code>BigDecimal</code> for the right hand side of
  *             the remainder operation.
  * @param  set The <code>MathContext</code> arithmetic settings.
  * @return     A <code>BigDecimal</code> whose value is the remainder
  *             of <code>this+rhs</code>.
  * @throws ArithmeticException if <code>rhs</code> is zero.
  * @throws ArithmeticException if the integer part of the result will
  *             not fit in the number of digits specified for the
  *             context.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal remainder(com.ibm.icu.math.BigDecimal rhs,com.ibm.icu.math.MathContext set){
 function remainder() {
  var set;
  if (remainder.arguments.length == 2)
   {
    set = remainder.arguments[1];
   }
  else if (remainder.arguments.length == 1)
   {
    set = this.plainMC;
   }
  else
   {
    throw "remainder(): " + remainder.arguments.length + " arguments given; expected 1 or 2";
   }
  var rhs = remainder.arguments[0];
  return this.dodivide('R',rhs,set,-1);
  }

 /**
  * Returns a plain <code>BigDecimal</code> whose value is
  * <code>this-rhs</code>, using fixed point arithmetic.
  * <p>
  * The same as {@link #subtract(BigDecimal, MathContext)},
  * where the <code>BigDecimal</code> is <code>rhs</code>,
  * and the context is <code>new MathContext(0, MathContext.PLAIN)</code>.
  * <p>
  * The length of the decimal part (the scale) of the result will be
  * the maximum of the scales of the two operands.
  *
  * @param  rhs The <code>BigDecimal</code> for the right hand side of
  *             the subtraction.
  * @return     A <code>BigDecimal</code> whose value is
  *             <code>this-rhs</code>, using fixed point arithmetic.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal subtract(com.ibm.icu.math.BigDecimal rhs){
 //-- return this.subtract(rhs,plainMC);
 //-- }

 /**
  * Returns a <code>BigDecimal</code> whose value is <code>this-rhs</code>.
  * <p>
  * Implements the subtraction (<b><code>-</code></b>) operator
  * (as defined in the decimal documentation, see {@link BigDecimal
  * class header}),
  * and returns the result as a <code>BigDecimal</code> object.
  *
  * @param  rhs The <code>BigDecimal</code> for the right hand side of
  *             the subtraction.
  * @param  set The <code>MathContext</code> arithmetic settings.
  * @return     A <code>BigDecimal</code> whose value is
  *             <code>this-rhs</code>.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal subtract(com.ibm.icu.math.BigDecimal rhs,com.ibm.icu.math.MathContext set){
 function subtract() {
  var set;
  if (subtract.arguments.length == 2)
   {
    set = subtract.arguments[1];
   }
  else if (subtract.arguments.length == 1)
   {
    set = this.plainMC;
   }
  else
   {
    throw "subtract(): " + subtract.arguments.length + " arguments given; expected 1 or 2";
   }
  var rhs = subtract.arguments[0];
  //--com.ibm.icu.math.BigDecimal newrhs;
  var newrhs;
  if (set.lostDigits)
   this.checkdigits(rhs,set.digits);
  // [add will recheck .. but would report -rhs]
  /* carry out the subtraction */
  // we could fastpath -0, but it is too rare.
  newrhs=this.clone(rhs); // safe copy
  newrhs.ind=-newrhs.ind; // prepare to subtract
  return this.add(newrhs,set); // arithmetic
  }

 /* ---------------------------------------------------------------- */
 /* Other methods                                                    */
 /* ---------------------------------------------------------------- */

 /**
  * Converts this <code>BigDecimal</code> to a <code>byte</code>.
  * If the <code>BigDecimal</code> has a non-zero decimal part or is
  * out of the possible range for a <code>byte</code> (8-bit signed
  * integer) result then an <code>ArithmeticException</code> is thrown.
  *
  * @return A <code>byte</code> equal in value to <code>this</code>.
  * @throws ArithmeticException if <code>this</code> has a non-zero
  *                 decimal part, or will not fit in a <code>byte</code>.
  * @stable ICU 2.0
  */

 //--public byte byteValueExact(){
 //-- int num;
 //-- num=this.intValueExact(); // will check decimal part too
 //-- if ((num>127)|(num<(-128)))
 //--  throw new java.lang.ArithmeticException("Conversion overflow:"+" "+this.toString());
 //-- return (byte)num;
 //-- }

 /**
  * Compares this <code>BigDecimal</code> with the value of the parameter.
  * <p>
  * If the parameter is <code>null</code>, or is not an instance of the
  * <code>BigDecimal</code> type, an exception is thrown.
  * Otherwise, the parameter is cast to type <code>BigDecimal</code>
  * and the result of the {@link #compareTo(BigDecimal)} method,
  * using the cast parameter, is returned.
  * <p>
  * The {@link #compareTo(BigDecimal, MathContext)} method should be
  * used when a <code>MathContext</code> is needed for the comparison.
  *
  * @param  rhs The <code>Object</code> for the right hand side of
  *             the comparison.
  * @return     An <code>int</code> whose value is -1, 0, or 1 as
  *             <code>this</code> is numerically less than, equal to,
  *             or greater than <code>rhs</code>.
  * @throws ClassCastException if <code>rhs</code> cannot be cast to
  *                 a <code>BigDecimal</code> object.
  * @see    #compareTo(BigDecimal)
  * @stable ICU 2.0
  */

 //--public int compareTo(java.lang.Object rhsobj){
 //-- // the cast in the next line will raise ClassCastException if necessary
 //-- return compareTo((com.ibm.icu.math.BigDecimal)rhsobj,plainMC);
 //-- }

 /**
  * Converts this <code>BigDecimal</code> to a <code>double</code>.
  * If the <code>BigDecimal</code> is out of the possible range for a
  * <code>double</code> (64-bit signed floating point) result then an
  * <code>ArithmeticException</code> is thrown.
  * <p>
  * The double produced is identical to result of expressing the
  * <code>BigDecimal</code> as a <code>String</code> and then
  * converting it using the <code>Double(String)</code> constructor;
  * this can result in values of <code>Double.NEGATIVE_INFINITY</code>
  * or <code>Double.POSITIVE_INFINITY</code>.
  *
  * @return A <code>double</code> corresponding to <code>this</code>.
  * @stable ICU 2.0
  */

 //--public double doubleValue(){
 //-- // We go via a String [as does BigDecimal in JDK 1.2]
 //-- // Next line could possibly raise NumberFormatException
 //-- return java.lang.Double.valueOf(this.toString()).doubleValue();
 //-- }

 /**
  * Compares this <code>BigDecimal</code> with <code>rhs</code> for
  * equality.
  * <p>
  * If the parameter is <code>null</code>, or is not an instance of the
  * BigDecimal type, or is not exactly equal to the current
  * <code>BigDecimal</code> object, then <i>false</i> is returned.
  * Otherwise, <i>true</i> is returned.
  * <p>
  * "Exactly equal", here, means that the <code>String</code>
  * representations of the <code>BigDecimal</code> numbers are
  * identical (they have the same characters in the same sequence).
  * <p>
  * The {@link #compareTo(BigDecimal, MathContext)} method should be
  * used for more general comparisons.
  * @param  rhs The <code>Object</code> for the right hand side of
  *             the comparison.
  * @return     A <code>boolean</code> whose value <i>true</i> if and
  *             only if the operands have identical string representations.
  * @throws ClassCastException if <code>rhs</code> cannot be cast to
  *                 a <code>BigDecimal</code> object.
  * @stable ICU 2.0
  * @see    #compareTo(Object)
  * @see    #compareTo(BigDecimal)
  * @see    #compareTo(BigDecimal, MathContext)
  */

 //--public boolean equals(java.lang.Object obj){
 function equals(obj) {
  //--com.ibm.icu.math.BigDecimal rhs;
  var rhs;
  //--int i=0;
  var i=0;
  //--char lca[]=null;
  var lca=null;
  //--char rca[]=null;
  var rca=null;
  // We are equal iff toString of both are exactly the same
  if (obj==null)
   return false; // not equal
  if ((!(((obj instanceof BigDecimal)))))
   return false; // not a decimal
  rhs=obj; // cast; we know it will work
  if (this.ind!=rhs.ind)
   return false; // different signs never match
  if (((this.mant.length==rhs.mant.length)&&(this.exp==rhs.exp))&&(this.form==rhs.form))

   { // mantissas say all
    // here with equal-length byte arrays to compare
    {var $8=this.mant.length;i=0;i:for(;$8>0;$8--,i++){
     if (this.mant[i]!=rhs.mant[i])
      return false;
     }
    }/*i*/
   }
  else
   { // need proper layout
    lca=this.layout(); // layout to character array
    rca=rhs.layout();
    if (lca.length!=rca.length)
     return false; // mismatch
    // here with equal-length character arrays to compare
    {var $9=lca.length;i=0;i:for(;$9>0;$9--,i++){
     if (lca[i]!=rca[i])
      return false;
     }
    }/*i*/
   }
  return true; // arrays have identical content
  }

 /**
  * Converts this <code>BigDecimal</code> to a <code>float</code>.
  * If the <code>BigDecimal</code> is out of the possible range for a
  * <code>float</code> (32-bit signed floating point) result then an
  * <code>ArithmeticException</code> is thrown.
  * <p>
  * The float produced is identical to result of expressing the
  * <code>BigDecimal</code> as a <code>String</code> and then
  * converting it using the <code>Float(String)</code> constructor;
  * this can result in values of <code>Float.NEGATIVE_INFINITY</code>
  * or <code>Float.POSITIVE_INFINITY</code>.
  *
  * @return A <code>float</code> corresponding to <code>this</code>.
  * @stable ICU 2.0
  */

 //--public float floatValue(){
 //-- return java.lang.Float.valueOf(this.toString()).floatValue();
 //-- }

 /**
  * Returns the <code>String</code> representation of this
  * <code>BigDecimal</code>, modified by layout parameters.
  * <p>
  * <i>This method is provided as a primitive for use by more
  * sophisticated classes, such as <code>DecimalFormat</code>, that
  * can apply locale-sensitive editing of the result.  The level of
  * formatting that it provides is a necessary part of the BigDecimal
  * class as it is sensitive to and must follow the calculation and
  * rounding rules for BigDecimal arithmetic.
  * However, if the function is provided elsewhere, it may be removed
  * from this class. </i>
  * <p>
  * The parameters, for both forms of the <code>format</code> method
  * are all of type <code>int</code>.
  * A value of -1 for any parameter indicates that the default action
  * or value for that parameter should be used.
  * <p>
  * The parameters, <code>before</code> and <code>after</code>,
  * specify the number of characters to be used for the integer part
  * and decimal part of the result respectively.  Exponential notation
  * is not used. If either parameter is -1 (which indicates the default
  * action), the number of characters used will be exactly as many as
  * are needed for that part.
  * <p>
  * <code>before</code> must be a positive number; if it is larger than
  * is needed to contain the integer part, that part is padded on the
  * left with blanks to the requested length. If <code>before</code> is
  * not large enough to contain the integer part of the number
  * (including the sign, for negative numbers) an exception is thrown.
  * <p>
  * <code>after</code> must be a non-negative number; if it is not the
  * same size as the decimal part of the number, the number will be
  * rounded (or extended with zeros) to fit.  Specifying 0 for
  * <code>after</code> will cause the number to be rounded to an
  * integer (that is, it will have no decimal part or decimal point).
  * The rounding method will be the default,
  * <code>MathContext.ROUND_HALF_UP</code>.
  * <p>
  * Other rounding methods, and the use of exponential notation, can
  * be selected by using {@link #format(int,int,int,int,int,int)}.
  * Using the two-parameter form of the method has exactly the same
  * effect as using the six-parameter form with the final four
  * parameters all being -1.
  *
  * @param  before The <code>int</code> specifying the number of places
  *                before the decimal point.  Use -1 for 'as many as
  *                are needed'.
  * @param  after  The <code>int</code> specifying the number of places
  *                after the decimal point.  Use -1 for 'as many as are
  *                needed'.
  * @return        A <code>String</code> representing this
  *                <code>BigDecimal</code>, laid out according to the
  *                specified parameters
  * @throws ArithmeticException if the number cannot be laid out as
  *                requested.
  * @throws IllegalArgumentException if a parameter is out of range.
  * @stable ICU 2.0
  * @see    #toString
  * @see    #toCharArray
  */

 //--public java.lang.String format(int before,int after){
 //-- return format(before,after,-1,-1,com.ibm.icu.math.MathContext.SCIENTIFIC,ROUND_HALF_UP);
 //-- }

 /**
  * Returns the <code>String</code> representation of this
  * <code>BigDecimal</code>, modified by layout parameters and allowing
  * exponential notation.
  * <p>
  * <i>This method is provided as a primitive for use by more
  * sophisticated classes, such as <code>DecimalFormat</code>, that
  * can apply locale-sensitive editing of the result.  The level of
  * formatting that it provides is a necessary part of the BigDecimal
  * class as it is sensitive to and must follow the calculation and
  * rounding rules for BigDecimal arithmetic.
  * However, if the function is provided elsewhere, it may be removed
  * from this class. </i>
  * <p>
  * The parameters are all of type <code>int</code>.
  * A value of -1 for any parameter indicates that the default action
  * or value for that parameter should be used.
  * <p>
  * The first two parameters (<code>before</code> and
  * <code>after</code>) specify the number of characters to be used for
  * the integer part and decimal part of the result respectively, as
  * defined for {@link #format(int,int)}.
  * If either of these is -1 (which indicates the default action), the
  * number of characters used will be exactly as many as are needed for
  * that part.
  * <p>
  * The remaining parameters control the use of exponential notation
  * and rounding.  Three (<code>explaces</code>, <code>exdigits</code>,
  * and <code>exform</code>) control the exponent part of the result.
  * As before, the default action for any of these parameters may be
  * selected by using the value -1.
  * <p>
  * <code>explaces</code> must be a positive number; it sets the number
  * of places (digits after the sign of the exponent) to be used for
  * any exponent part, the default (when <code>explaces</code> is -1)
  * being to use as many as are needed.
  * If <code>explaces</code> is not -1, space is always reserved for
  * an exponent; if one is not needed (for example, if the exponent
  * will be 0) then <code>explaces</code>+2 blanks are appended to the
  * result.
  * <!-- (This preserves vertical alignment of similarly formatted
  *       numbers in a monospace font.) -->
  * If <code>explaces</code> is not -1 and is not large enough to
  * contain the exponent, an exception is thrown.
  * <p>
  * <code>exdigits</code> sets the trigger point for use of exponential
  * notation. If, before any rounding, the number of places needed
  * before the decimal point exceeds <code>exdigits</code>, or if the
  * absolute value of the result is less than <code>0.000001</code>,
  * then exponential form will be used, provided that
  * <code>exdigits</code> was specified.
  * When <code>exdigits</code> is -1, exponential notation will never
  * be used. If 0 is specified for <code>exdigits</code>, exponential
  * notation is always used unless the exponent would be 0.
  * <p>
  * <code>exform</code> sets the form for exponential notation (if
  * needed).
  * It  may be either {@link MathContext#SCIENTIFIC} or
  * {@link MathContext#ENGINEERING}.
  * If the latter, engineering, form is requested, up to three digits
  * (plus sign, if negative) may be needed for the integer part of the
  * result (<code>before</code>).  Otherwise, only one digit (plus
  * sign, if negative) is needed.
  * <p>
  * Finally, the sixth argument, <code>exround</code>, selects the
  * rounding algorithm to be used, and must be one of the values
  * indicated by a public constant in the {@link MathContext} class
  * whose name starts with <code>ROUND_</code>.
  * The default (<code>ROUND_HALF_UP</code>) may also be selected by
  * using the value -1, as before.
  * <p>
  * The special value <code>MathContext.ROUND_UNNECESSARY</code> may be
  * used to detect whether non-zero digits are discarded -- if
  * <code>exround</code> has this value than if non-zero digits would
  * be discarded (rounded) during formatting then an
  * <code>ArithmeticException</code> is thrown.
  *
  * @param  before   The <code>int</code> specifying the number of places
  *                  before the decimal point.
  *                  Use -1 for 'as many as are needed'.
  * @param  after    The <code>int</code> specifying the number of places
  *                  after the decimal point.
  *                  Use -1 for 'as many as are needed'.
  * @param  explaces The <code>int</code> specifying the number of places
  *                  to be used for any exponent.
  *                  Use -1 for 'as many as are needed'.
  * @param  exdigits The <code>int</code> specifying the trigger
  *                  (digits before the decimal point) which if
  *                  exceeded causes exponential notation to be used.
  *                  Use 0 to force exponential notation.
  *                  Use -1 to force plain notation (no exponential
  *                  notation).
  * @param  exform   The <code>int</code> specifying the form of
  *                  exponential notation to be used
  *                  ({@link MathContext#SCIENTIFIC} or
  *                  {@link MathContext#ENGINEERING}).
  * @param  exround  The <code>int</code> specifying the rounding mode
  *                  to use.
  *                  Use -1 for the default, {@link MathContext#ROUND_HALF_UP}.
  * @return          A <code>String</code> representing this
  *                  <code>BigDecimal</code>, laid out according to the
  *                  specified parameters
  * @throws ArithmeticException if the number cannot be laid out as
  *                  requested.
  * @throws IllegalArgumentException if a parameter is out of range.
  * @see    #toString
  * @see    #toCharArray
  * @stable ICU 2.0
  */

 //--public java.lang.String format(int before,int after,int explaces,int exdigits,int exformint,int exround){
 function format() {
  var explaces;
  var exdigits;
  var exformint;
  var exround;
  if (format.arguments.length == 6)
   {
    explaces = format.arguments[2];
    exdigits = format.arguments[3];
    exformint = format.arguments[4];
    exround = format.arguments[5];
   }
  else if (format.arguments.length == 2)
   {
    explaces = -1;
    exdigits = -1;
    exformint = MathContext.prototype.SCIENTIFIC;
    exround = this.ROUND_HALF_UP;
   }
  else
   {
    throw "format(): " + format.arguments.length + " arguments given; expected 2 or 6";
   }
  var before = format.arguments[0];
  var after = format.arguments[1];
  //--com.ibm.icu.math.BigDecimal num;
  var num;
  //--int mag=0;
  var mag=0;
  //--int thisafter=0;
  var thisafter=0;
  //--int lead=0;
  var lead=0;
  //--byte newmant[]=null;
  var newmant=null;
  //--int chop=0;
  var chop=0;
  //--int need=0;
  var need=0;
  //--int oldexp=0;
  var oldexp=0;
  //--char a[];
  var a;
  //--int p=0;
  var p=0;
  //--char newa[]=null;
  var newa=null;
  //--int i=0;
  var i=0;
  //--int places=0;
  var places=0;


  /* Check arguments */
  if ((before<(-1))||(before==0))
   this.badarg("format",1,before);
  if (after<(-1))
   this.badarg("format",2,after);
  if ((explaces<(-1))||(explaces==0))
   this.badarg("format",3,explaces);
  if (exdigits<(-1))
   this.badarg("format",4,exdigits);
  {/*select*/
  if (exformint==MathContext.prototype.SCIENTIFIC)
   {}
  else if (exformint==MathContext.prototype.ENGINEERING)
   {}
  else if (exformint==(-1))
   exformint=MathContext.prototype.SCIENTIFIC;
   // note PLAIN isn't allowed
  else{
   this.badarg("format",5,exformint);
  }
  }
  // checking the rounding mode is done by trying to construct a
  // MathContext object with that mode; it will fail if bad
  if (exround!=this.ROUND_HALF_UP)
   {try{ // if non-default...
    if (exround==(-1))
     exround=this.ROUND_HALF_UP;
    else
     new MathContext(9,MathContext.prototype.SCIENTIFIC,false,exround);
   }
   catch ($10){
    this.badarg("format",6,exround);
   }}

  num=this.clone(this); // make private copy

  /* Here:
     num       is BigDecimal to format
     before    is places before point [>0]
     after     is places after point  [>=0]
     explaces  is exponent places     [>0]
     exdigits  is exponent digits     [>=0]
     exformint is exponent form       [one of two]
     exround   is rounding mode       [one of eight]
     'before' through 'exdigits' are -1 if not specified
  */

  /* determine form */
  {setform:do{/*select*/
  if (exdigits==(-1))
   num.form=MathContext.prototype.PLAIN;
  else if (num.ind==this.iszero)
   num.form=MathContext.prototype.PLAIN;
  else{
   // determine whether triggers
   mag=num.exp+num.mant.length;
   if (mag>exdigits)
    num.form=exformint;
   else
    if (mag<(-5))
     num.form=exformint;
    else
     num.form=MathContext.prototype.PLAIN;
  }
  }while(false);}/*setform*/

  /* If 'after' was specified then we may need to adjust the
     mantissa.  This is a little tricky, as we must conform to the
     rules of exponential layout if necessary (e.g., we cannot end up
     with 10.0 if scientific). */
  if (after>=0)
   {setafter:for(;;){
    // calculate the current after-length
    {/*select*/
    if (num.form==MathContext.prototype.PLAIN)
     thisafter=-num.exp; // has decimal part
    else if (num.form==MathContext.prototype.SCIENTIFIC)
     thisafter=num.mant.length-1;
    else{ // engineering
     lead=(((num.exp+num.mant.length)-1))%3; // exponent to use
     if (lead<0)
      lead=3+lead; // negative exponent case
     lead++; // number of leading digits
     if (lead>=num.mant.length)
      thisafter=0;
     else
      thisafter=num.mant.length-lead;
    }
    }
    if (thisafter==after)
     break setafter; // we're in luck
    if (thisafter<after)
     { // need added trailing zeros
      // [thisafter can be negative]
      newmant=this.extend(num.mant,(num.mant.length+after)-thisafter);
      num.mant=newmant;
      num.exp=num.exp-((after-thisafter)); // adjust exponent
      if (num.exp<this.MinExp)
       throw "format(): Exponent Overflow: " + num.exp;
      break setafter;
     }
    // We have too many digits after the decimal point; this could
    // cause a carry, which could change the mantissa...
    // Watch out for implied leading zeros in PLAIN case
    chop=thisafter-after; // digits to lop [is >0]
    if (chop>num.mant.length)
     { // all digits go, no chance of carry
      // carry on with zero
      num.mant=this.ZERO.mant;
      num.ind=this.iszero;
      num.exp=0;
      continue setafter; // recheck: we may need trailing zeros
     }
    // we have a digit to inspect from existing mantissa
    // round the number as required
    need=num.mant.length-chop; // digits to end up with [may be 0]
    oldexp=num.exp; // save old exponent
    num.round(need,exround);
    // if the exponent grew by more than the digits we chopped, then
    // we must have had a carry, so will need to recheck the layout
    if ((num.exp-oldexp)==chop)
     break setafter; // number did not have carry
    // mantissa got extended .. so go around and check again
    }
   }/*setafter*/

  a=num.layout(); // lay out, with exponent if required, etc.

  /* Here we have laid-out number in 'a' */
  // now apply 'before' and 'explaces' as needed
  if (before>0)
   {
    // look for '.' or 'E'
    {var $11=a.length;p=0;p:for(;$11>0;$11--,p++){
     if (a[p]=='.')
      break p;
     if (a[p]=='E')
      break p;
     }
    }/*p*/
    // p is now offset of '.', 'E', or character after end of array
    // that is, the current length of before part
    if (p>before)
     this.badarg("format",1,before); // won't fit
    if (p<before)
     { // need leading blanks
      newa=new Array((a.length+before)-p);
      {var $12=before-p;i=0;i:for(;$12>0;$12--,i++){
       newa[i]=' ';
       }
      }/*i*/
      //--java.lang.System.arraycopy((java.lang.Object)a,0,(java.lang.Object)newa,i,a.length);
      this.arraycopy(a,0,newa,i,a.length);
      a=newa;
     }
   // [if p=before then it's just the right length]
   }

  if (explaces>0)
   {
    // look for 'E' [cannot be at offset 0]
    {var $13=a.length-1;p=a.length-1;p:for(;$13>0;$13--,p--){
     if (a[p]=='E')
      break p;
     }
    }/*p*/
    // p is now offset of 'E', or 0
    if (p==0)
     { // no E part; add trailing blanks
      newa=new Array((a.length+explaces)+2);
      //--java.lang.System.arraycopy((java.lang.Object)a,0,(java.lang.Object)newa,0,a.length);
      this.arraycopy(a,0,newa,0,a.length);
      {var $14=explaces+2;i=a.length;i:for(;$14>0;$14--,i++){
       newa[i]=' ';
       }
      }/*i*/
      a=newa;
     }
    else
     {/* found E */ // may need to insert zeros
      places=(a.length-p)-2; // number so far
      if (places>explaces)
       this.badarg("format",3,explaces);
      if (places<explaces)
       { // need to insert zeros
        newa=new Array((a.length+explaces)-places);
        //--java.lang.System.arraycopy((java.lang.Object)a,0,(java.lang.Object)newa,0,p+2); // through E and sign
        this.arraycopy(a,0,newa,0,p+2);
        {var $15=explaces-places;i=p+2;i:for(;$15>0;$15--,i++){
         newa[i]='0';
         }
        }/*i*/
        //--java.lang.System.arraycopy((java.lang.Object)a,p+2,(java.lang.Object)newa,i,places); // remainder of exponent
        this.arraycopy(a,p+2,newa,i,places);
        a=newa;
       }
     // [if places=explaces then it's just the right length]
     }
   }
  return a.join("");
  }

 /**
  * Returns the hashcode for this <code>BigDecimal</code>.
  * This hashcode is suitable for use by the
  * <code>java.util.Hashtable</code> class.
  * <p>
  * Note that two <code>BigDecimal</code> objects are only guaranteed
  * to produce the same hashcode if they are exactly equal (that is,
  * the <code>String</code> representations of the
  * <code>BigDecimal</code> numbers are identical -- they have the same
  * characters in the same sequence).
  *
  * @return An <code>int</code> that is the hashcode for <code>this</code>.
  * @stable ICU 2.0
  */

 //--public int hashCode(){
 //-- // Maybe calculate ourselves, later.  If so, note that there can be
 //-- // more than one internal representation for a given toString() result.
 //-- return this.toString().hashCode();
 //-- }

 /**
  * Converts this <code>BigDecimal</code> to an <code>int</code>.
  * If the <code>BigDecimal</code> has a non-zero decimal part it is
  * discarded. If the <code>BigDecimal</code> is out of the possible
  * range for an <code>int</code> (32-bit signed integer) result then
  * only the low-order 32 bits are used. (That is, the number may be
  * <i>decapitated</i>.)  To avoid unexpected errors when these
  * conditions occur, use the {@link #intValueExact} method.
  *
  * @return An <code>int</code> converted from <code>this</code>,
  *         truncated and decapitated if necessary.
  * @stable ICU 2.0
  */

 //--public int intValue(){
 //-- return toBigInteger().intValue();
 //-- }

 /**
  * Converts this <code>BigDecimal</code> to an <code>int</code>.
  * If the <code>BigDecimal</code> has a non-zero decimal part or is
  * out of the possible range for an <code>int</code> (32-bit signed
  * integer) result then an <code>ArithmeticException</code> is thrown.
  *
  * @return An <code>int</code> equal in value to <code>this</code>.
  * @throws ArithmeticException if <code>this</code> has a non-zero
  *                 decimal part, or will not fit in an
  *                 <code>int</code>.
  * @stable ICU 2.0
  */

 //--public int intValueExact(){
 function intValueExact() {
  //--int lodigit;
  var lodigit;
  //--int useexp=0;
  var useexp=0;
  //--int result;
  var result;
  //--int i=0;
  var i=0;
  //--int topdig=0;
  var topdig=0;
  // This does not use longValueExact() as the latter can be much
  // slower.
  // intcheck (from pow) relies on this to check decimal part
  if (this.ind==this.iszero)
   return 0; // easy, and quite common
  /* test and drop any trailing decimal part */
  lodigit=this.mant.length-1;
  if (this.exp<0)
   {
    lodigit=lodigit+this.exp; // reduces by -(-exp)
    /* all decimal places must be 0 */
    if ((!(this.allzero(this.mant,lodigit+1))))
     throw "intValueExact(): Decimal part non-zero: " + this.toString();
    if (lodigit<0)
     return 0; // -1<this<1
    useexp=0;
   }
  else
   {/* >=0 */
    if ((this.exp+lodigit)>9)  // early exit
     throw "intValueExact(): Conversion overflow: "+this.toString();
    useexp=this.exp;
   }
  /* convert the mantissa to binary, inline for speed */
  result=0;
  {var $16=lodigit+useexp;i=0;i:for(;i<=$16;i++){
   result=result*10;
   if (i<=lodigit)
    result=result+this.mant[i];
   }
  }/*i*/

  /* Now, if the risky length, check for overflow */
  if ((lodigit+useexp)==9)
   {
    // note we cannot just test for -ve result, as overflow can move a
    // zero into the top bit [consider 5555555555]
    topdig=div(result,1000000000); // get top digit, preserving sign
    if (topdig!=this.mant[0])
     { // digit must match and be positive
      // except in the special case ...
      if (result==-2147483648)  // looks like the special
       if (this.ind==this.isneg)  // really was negative
        if (this.mant[0]==2)
         return result; // really had top digit 2
      throw "intValueExact(): Conversion overflow: "+this.toString();
     }
   }

  /* Looks good */
  if (this.ind==this.ispos)
   return result;
  return -result;
  }

 /**
  * Converts this <code>BigDecimal</code> to a <code>long</code>.
  * If the <code>BigDecimal</code> has a non-zero decimal part it is
  * discarded. If the <code>BigDecimal</code> is out of the possible
  * range for a <code>long</code> (64-bit signed integer) result then
  * only the low-order 64 bits are used. (That is, the number may be
  * <i>decapitated</i>.)  To avoid unexpected errors when these
  * conditions occur, use the {@link #longValueExact} method.
  *
  * @return A <code>long</code> converted from <code>this</code>,
  *         truncated and decapitated if necessary.
  * @stable ICU 2.0
  */

 //--public long longValue(){
 //-- return toBigInteger().longValue();
 //-- }

 /**
  * Converts this <code>BigDecimal</code> to a <code>long</code>.
  * If the <code>BigDecimal</code> has a non-zero decimal part or is
  * out of the possible range for a <code>long</code> (64-bit signed
  * integer) result then an <code>ArithmeticException</code> is thrown.
  *
  * @return A <code>long</code> equal in value to <code>this</code>.
  * @throws ArithmeticException if <code>this</code> has a non-zero
  *                 decimal part, or will not fit in a
  *                 <code>long</code>.
  * @stable ICU 2.0
  */

 //--public long longValueExact(){
 //-- int lodigit;
 //-- int cstart=0;
 //-- int useexp=0;
 //-- long result;
 //-- int i=0;
 //-- long topdig=0;
 //-- // Identical to intValueExact except for result=long, and exp>=20 test
 //-- if (ind==0)
 //--  return 0; // easy, and quite common
 //-- lodigit=mant.length-1; // last included digit
 //-- if (exp<0)
 //--  {
 //--   lodigit=lodigit+exp; // -(-exp)
 //--   /* all decimal places must be 0 */
 //--   if (lodigit<0)
 //--    cstart=0;
 //--   else
 //--    cstart=lodigit+1;
 //--   if ((!(allzero(mant,cstart))))
 //--    throw new java.lang.ArithmeticException("Decimal part non-zero:"+" "+this.toString());
 //--   if (lodigit<0)
 //--    return 0; // -1<this<1
 //--   useexp=0;
 //--  }
 //-- else
 //--  {/* >=0 */
 //--   if ((exp+mant.length)>18)  // early exit
 //--    throw new java.lang.ArithmeticException("Conversion overflow:"+" "+this.toString());
 //--   useexp=exp;
 //--  }
 //--
 //-- /* convert the mantissa to binary, inline for speed */
 //-- // note that we could safely use the 'test for wrap to negative'
 //-- // algorithm here, but instead we parallel the intValueExact
 //-- // algorithm for ease of checking and maintenance.
 //-- result=(long)0;
 //-- {int $17=lodigit+useexp;i=0;i:for(;i<=$17;i++){
 //--  result=result*10;
 //--  if (i<=lodigit)
 //--   result=result+mant[i];
 //--  }
 //-- }/*i*/
 //--
 //-- /* Now, if the risky length, check for overflow */
 //-- if ((lodigit+useexp)==18)
 //--  {
 //--   topdig=result/1000000000000000000L; // get top digit, preserving sign
 //--   if (topdig!=mant[0])
 //--    { // digit must match and be positive
 //--     // except in the special case ...
 //--     if (result==java.lang.Long.MIN_VALUE)  // looks like the special
 //--      if (ind==isneg)  // really was negative
 //--       if (mant[0]==9)
 //--        return result; // really had top digit 9
 //--     throw new java.lang.ArithmeticException("Conversion overflow:"+" "+this.toString());
 //--    }
 //--  }
 //--
 //-- /* Looks good */
 //-- if (ind==ispos)
 //--  return result;
 //-- return (long)-result;
 //-- }

 /**
  * Returns a plain <code>BigDecimal</code> whose decimal point has
  * been moved to the left by a specified number of positions.
  * The parameter, <code>n</code>, specifies the number of positions to
  * move the decimal point.
  * That is, if <code>n</code> is 0 or positive, the number returned is
  * given by:
  * <p><code>
  * this.multiply(TEN.pow(new BigDecimal(-n)))
  * </code>
  * <p>
  * <code>n</code> may be negative, in which case the method returns
  * the same result as <code>movePointRight(-n)</code>.
  *
  * @param  n The <code>int</code> specifying the number of places to
  *           move the decimal point leftwards.
  * @return   A <code>BigDecimal</code> derived from
  *           <code>this</code>, with the decimal point moved
  *           <code>n</code> places to the left.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal movePointLeft(int n){
 function movePointLeft(n) {
  //--com.ibm.icu.math.BigDecimal res;
  var res;
  // very little point in optimizing for shift of 0
  res=this.clone(this);
  res.exp=res.exp-n;
  return res.finish(this.plainMC,false); // finish sets form and checks exponent
  }

 /**
  * Returns a plain <code>BigDecimal</code> whose decimal point has
  * been moved to the right by a specified number of positions.
  * The parameter, <code>n</code>, specifies the number of positions to
  * move the decimal point.
  * That is, if <code>n</code> is 0 or positive, the number returned is
  * given by:
  * <p><code>
  * this.multiply(TEN.pow(new BigDecimal(n)))
  * </code>
  * <p>
  * <code>n</code> may be negative, in which case the method returns
  * the same result as <code>movePointLeft(-n)</code>.
  *
  * @param  n The <code>int</code> specifying the number of places to
  *           move the decimal point rightwards.
  * @return   A <code>BigDecimal</code> derived from
  *           <code>this</code>, with the decimal point moved
  *           <code>n</code> places to the right.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal movePointRight(int n){
 function movePointRight(n) {
  //--com.ibm.icu.math.BigDecimal res;
  var res;
  res=this.clone(this);
  res.exp=res.exp+n;
  return res.finish(this.plainMC,false);
  }

 /**
  * Returns the scale of this <code>BigDecimal</code>.
  * Returns a non-negative <code>int</code> which is the scale of the
  * number. The scale is the number of digits in the decimal part of
  * the number if the number were formatted without exponential
  * notation.
  *
  * @return An <code>int</code> whose value is the scale of this
  *         <code>BigDecimal</code>.
  * @stable ICU 2.0
  */

 //--public int scale(){
 function scale() {
  if (this.exp>=0)
   return 0; // scale can never be negative
  return -this.exp;
  }

 /**
  * Returns a plain <code>BigDecimal</code> with a given scale.
  * <p>
  * If the given scale (which must be zero or positive) is the same as
  * or greater than the length of the decimal part (the scale) of this
  * <code>BigDecimal</code> then trailing zeros will be added to the
  * decimal part as necessary.
  * <p>
  * If the given scale is less than the length of the decimal part (the
  * scale) of this <code>BigDecimal</code> then trailing digits
  * will be removed, and in this case an
  * <code>ArithmeticException</code> is thrown if any discarded digits
  * are non-zero.
  * <p>
  * The same as {@link #setScale(int, int)}, where the first parameter
  * is the scale, and the second is
  * <code>MathContext.ROUND_UNNECESSARY</code>.
  *
  * @param  scale The <code>int</code> specifying the scale of the
  *               resulting <code>BigDecimal</code>.
  * @return       A plain <code>BigDecimal</code> with the given scale.
  * @throws ArithmeticException if <code>scale</code> is negative.
  * @throws ArithmeticException if reducing scale would discard
  *               non-zero digits.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal setScale(int scale){
 //-- return setScale(scale,ROUND_UNNECESSARY);
 //-- }

 /**
  * Returns a plain <code>BigDecimal</code> with a given scale.
  * <p>
  * If the given scale (which must be zero or positive) is the same as
  * or greater than the length of the decimal part (the scale) of this
  * <code>BigDecimal</code> then trailing zeros will be added to the
  * decimal part as necessary.
  * <p>
  * If the given scale is less than the length of the decimal part (the
  * scale) of this <code>BigDecimal</code> then trailing digits
  * will be removed, and the rounding mode given by the second
  * parameter is used to determine if the remaining digits are
  * affected by a carry.
  * In this case, an <code>IllegalArgumentException</code> is thrown if
  * <code>round</code> is not a valid rounding mode.
  * <p>
  * If <code>round</code> is <code>MathContext.ROUND_UNNECESSARY</code>,
  * an <code>ArithmeticException</code> is thrown if any discarded
  * digits are non-zero.
  *
  * @param  scale The <code>int</code> specifying the scale of the
  *               resulting <code>BigDecimal</code>.
  * @param  round The <code>int</code> rounding mode to be used for
  *               the division (see the {@link MathContext} class).
  * @return       A plain <code>BigDecimal</code> with the given scale.
  * @throws IllegalArgumentException if <code>round</code> is not a
  *               valid rounding mode.
  * @throws ArithmeticException if <code>scale</code> is negative.
  * @throws ArithmeticException if <code>round</code> is
  *               <code>MathContext.ROUND_UNNECESSARY</code>, and
  *               reducing scale would discard non-zero digits.
  * @stable ICU 2.0
  */

 //--public com.ibm.icu.math.BigDecimal setScale(int scale,int round){
 function setScale() {
  var round;
  if (setScale.arguments.length == 2)
   {
    round = setScale.arguments[1];
   }
  else if (setScale.arguments.length == 1)
   {
    round = this.ROUND_UNNECESSARY;
   }
  else
   {
    throw "setScale(): " + setScale.arguments.length + " given; expected 1 or 2";
   }
  var scale = setScale.arguments[0];
  //--int ourscale;
  var ourscale;
  //--com.ibm.icu.math.BigDecimal res;
  var res;
  //--int padding=0;
  var padding=0;
  //--int newlen=0;
  var newlen=0;
  // at present this naughtily only checks the round value if it is
  // needed (used), for speed
  ourscale=this.scale();
  if (ourscale==scale)  // already correct scale
   if (this.form==MathContext.prototype.PLAIN)  // .. and form
    return this;
  res=this.clone(this); // need copy
  if (ourscale<=scale)
   { // simply zero-padding/changing form
    // if ourscale is 0 we may have lots of 0s to add
    if (ourscale==0)
     padding=res.exp+scale;
    else
     padding=scale-ourscale;
    res.mant=this.extend(res.mant,res.mant.length+padding);
    res.exp=-scale; // as requested
   }
  else
   {/* ourscale>scale: shortening, probably */
    if (scale<0)
     //--throw new java.lang.ArithmeticException("Negative scale:"+" "+scale);
     throw "setScale(): Negative scale: " + scale;
    // [round() will raise exception if invalid round]
    newlen=res.mant.length-((ourscale-scale)); // [<=0 is OK]
    res=res.round(newlen,round); // round to required length
    // This could have shifted left if round (say) 0.9->1[.0]
    // Repair if so by adding a zero and reducing exponent
    if (res.exp!=(-scale))
     {
      res.mant=this.extend(res.mant,res.mant.length+1);
      res.exp=res.exp-1;
     }
   }
  res.form=MathContext.prototype.PLAIN; // by definition
  return res;
  }

 /**
  * Converts this <code>BigDecimal</code> to a <code>short</code>.
  * If the <code>BigDecimal</code> has a non-zero decimal part or is
  * out of the possible range for a <code>short</code> (16-bit signed
  * integer) result then an <code>ArithmeticException</code> is thrown.
  *
  * @return A <code>short</code> equal in value to <code>this</code>.
  * @throws ArithmeticException if <code>this</code> has a non-zero
  *                 decimal part, or will not fit in a
  *                 <code>short</code>.
  * @stable ICU 2.0
  */

 //--public short shortValueExact(){
 //-- int num;
 //-- num=this.intValueExact(); // will check decimal part too
 //-- if ((num>32767)|(num<(-32768)))
 //--  throw new java.lang.ArithmeticException("Conversion overflow:"+" "+this.toString());
 //-- return (short)num;
 //-- }

 /**
  * Returns the sign of this <code>BigDecimal</code>, as an
  * <code>int</code>.
  * This returns the <i>signum</i> function value that represents the
  * sign of this <code>BigDecimal</code>.
  * That is, -1 if the <code>BigDecimal</code> is negative, 0 if it is
  * numerically equal to zero, or 1 if it is positive.
  *
  * @return An <code>int</code> which is -1 if the
  *         <code>BigDecimal</code> is negative, 0 if it is
  *         numerically equal to zero, or 1 if it is positive.
  * @stable ICU 2.0
  */

 //--public int signum(){
 function signum() {
  return this.ind; // [note this assumes values for ind.]
  }

 /**
  * Converts this <code>BigDecimal</code> to a
  * <code>java.math.BigDecimal</code>.
  * <p>
  * This is an exact conversion; the result is the same as if the
  * <code>BigDecimal</code> were formatted as a plain number without
  * any rounding or exponent and then the
  * <code>java.math.BigDecimal(java.lang.String)</code> constructor
  * were used to construct the result.
  * <p>
  * <i>(Note: this method is provided only in the
  * <code>com.ibm.icu.math</code> version of the BigDecimal class.
  * It would not be present in a <code>java.math</code> version.)</i>
  *
  * @return The <code>java.math.BigDecimal</code> equal in value
  *         to this <code>BigDecimal</code>.
  * @stable ICU 2.0
  */

 //--public java.math.BigDecimal toBigDecimal(){
 //-- return new java.math.BigDecimal(this.unscaledValue(),this.scale());
 //-- }

 /**
  * Converts this <code>BigDecimal</code> to a
  * <code>java.math.BigInteger</code>.
  * <p>
  * Any decimal part is truncated (discarded).
  * If an exception is desired should the decimal part be non-zero,
  * use {@link #toBigIntegerExact()}.
  *
  * @return The <code>java.math.BigInteger</code> equal in value
  *         to the integer part of this <code>BigDecimal</code>.
  * @stable ICU 2.0
  */

 //--public java.math.BigInteger toBigInteger(){
 //-- com.ibm.icu.math.BigDecimal res=null;
 //-- int newlen=0;
 //-- byte newmant[]=null;
 //-- {/*select*/
 //-- if ((exp>=0)&(form==com.ibm.icu.math.MathContext.PLAIN))
 //--  res=this; // can layout simply
 //-- else if (exp>=0)
 //--  {
 //--   res=clone(this); // safe copy
 //--   res.form=(byte)com.ibm.icu.math.MathContext.PLAIN; // .. and request PLAIN
 //--  }
 //-- else{
 //--  { // exp<0; scale to be truncated
 //--   // we could use divideInteger, but we may as well be quicker
 //--   if (((int)-this.exp)>=this.mant.length)
 //--    res=ZERO; // all blows away
 //--   else
 //--    {
 //--     res=clone(this); // safe copy
 //--     newlen=res.mant.length+res.exp;
 //--     newmant=new byte[newlen]; // [shorter]
 //--     java.lang.System.arraycopy((java.lang.Object)res.mant,0,(java.lang.Object)newmant,0,newlen);
 //--     res.mant=newmant;
 //--     res.form=(byte)com.ibm.icu.math.MathContext.PLAIN;
 //--     res.exp=0;
 //--    }
 //--  }
 //-- }
 //-- }
 //-- return new BigInteger(new java.lang.String(res.layout()));
 //-- }

 /**
  * Converts this <code>BigDecimal</code> to a
  * <code>java.math.BigInteger</code>.
  * <p>
  * An exception is thrown if the decimal part (if any) is non-zero.
  *
  * @return The <code>java.math.BigInteger</code> equal in value
  *         to the integer part of this <code>BigDecimal</code>.
  * @throws ArithmeticException if <code>this</code> has a non-zero
  *         decimal part.
  * @stable ICU 2.0
  */

 //--public java.math.BigInteger toBigIntegerExact(){
 //-- /* test any trailing decimal part */
 //-- if (exp<0)
 //--  { // possible decimal part
 //--   /* all decimal places must be 0; note exp<0 */
 //--   if ((!(allzero(mant,mant.length+exp))))
 //--    throw new java.lang.ArithmeticException("Decimal part non-zero:"+" "+this.toString());
 //--  }
 //-- return toBigInteger();
 //-- }

 /**
  * Returns the <code>BigDecimal</code> as a character array.
  * The result of this method is the same as using the
  * sequence <code>toString().toCharArray()</code>, but avoids creating
  * the intermediate <code>String</code> and <code>char[]</code>
  * objects.
  *
  * @return The <code>char[]</code> array corresponding to this
  *         <code>BigDecimal</code>.
  * @stable ICU 2.0
  */

 //--public char[] toCharArray(){
 //-- return layout();
 //-- }

 /**
  * Returns the <code>BigDecimal</code> as a <code>String</code>.
  * This returns a <code>String</code> that exactly represents this
  * <code>BigDecimal</code>, as defined in the decimal documentation
  * (see {@link BigDecimal class header}).
  * <p>
  * By definition, using the {@link #BigDecimal(String)} constructor
  * on the result <code>String</code> will create a
  * <code>BigDecimal</code> that is exactly equal to the original
  * <code>BigDecimal</code>.
  *
  * @return The <code>String</code> exactly corresponding to this
  *         <code>BigDecimal</code>.
  * @see    #format(int, int)
  * @see    #format(int, int, int, int, int, int)
  * @see    #toCharArray()
  * @stable ICU 2.0
  */

 //--public java.lang.String toString(){
 function toString() {
  return this.layout().join("");
  }

 /**
  * Returns the number as a <code>BigInteger</code> after removing the
  * scale.
  * That is, the number is expressed as a plain number, any decimal
  * point is then removed (retaining the digits of any decimal part),
  * and the result is then converted to a <code>BigInteger</code>.
  *
  * @return The <code>java.math.BigInteger</code> equal in value to
  *         this <code>BigDecimal</code> multiplied by ten to the
  *         power of <code>this.scale()</code>.
  * @stable ICU 2.0
  */

 //--public java.math.BigInteger unscaledValue(){
 //-- com.ibm.icu.math.BigDecimal res=null;
 //-- if (exp>=0)
 //--  res=this;
 //-- else
 //--  {
 //--   res=clone(this); // safe copy
 //--   res.exp=0; // drop scale
 //--  }
 //-- return res.toBigInteger();
 //-- }

 /**
  * Translates a <code>double</code> to a <code>BigDecimal</code>.
  * <p>
  * Returns a <code>BigDecimal</code> which is the decimal
  * representation of the 64-bit signed binary floating point
  * parameter. If the parameter is infinite, or is not a number (NaN),
  * a <code>NumberFormatException</code> is thrown.
  * <p>
  * The number is constructed as though <code>num</code> had been
  * converted to a <code>String</code> using the
  * <code>Double.toString()</code> method and the
  * {@link #BigDecimal(java.lang.String)} constructor had then been used.
  * This is typically not an exact conversion.
  *
  * @param  dub The <code>double</code> to be translated.
  * @return     The <code>BigDecimal</code> equal in value to
  *             <code>dub</code>.
  * @throws NumberFormatException if the parameter is infinite or
  *             not a number.
  * @stable ICU 2.0
  */

 //--public static com.ibm.icu.math.BigDecimal valueOf(double dub){
 //-- // Reminder: a zero double returns '0.0', so we cannot fastpath to
 //-- // use the constant ZERO.  This might be important enough to justify
 //-- // a factory approach, a cache, or a few private constants, later.
 //-- return new com.ibm.icu.math.BigDecimal((new java.lang.Double(dub)).toString());
 //-- }

 /**
  * Translates a <code>long</code> to a <code>BigDecimal</code>.
  * That is, returns a plain <code>BigDecimal</code> whose value is
  * equal to the given <code>long</code>.
  *
  * @param  lint The <code>long</code> to be translated.
  * @return      The <code>BigDecimal</code> equal in value to
  *              <code>lint</code>.
  * @stable ICU 2.0
  */

 //--public static com.ibm.icu.math.BigDecimal valueOf(long lint){
 //-- return valueOf(lint,0);
 //-- }

 /**
  * Translates a <code>long</code> to a <code>BigDecimal</code> with a
  * given scale.
  * That is, returns a plain <code>BigDecimal</code> whose unscaled
  * value is equal to the given <code>long</code>, adjusted by the
  * second parameter, <code>scale</code>.
  * <p>
  * The result is given by:
  * <p><code>
  * (new BigDecimal(lint)).divide(TEN.pow(new BigDecimal(scale)))
  * </code>
  * <p>
  * A <code>NumberFormatException</code> is thrown if <code>scale</code>
  * is negative.
  *
  * @param  lint  The <code>long</code> to be translated.
  * @param  scale The <code>int</code> scale to be applied.
  * @return       The <code>BigDecimal</code> equal in value to
  *               <code>lint</code>.
  * @throws NumberFormatException if the scale is negative.
  * @stable ICU 2.0
  */

 //--public static com.ibm.icu.math.BigDecimal valueOf(long lint,int scale){
 //-- com.ibm.icu.math.BigDecimal res=null;
 //-- {/*select*/
 //-- if (lint==0)
 //--  res=ZERO;
 //-- else if (lint==1)
 //--  res=ONE;
 //-- else if (lint==10)
 //--  res=TEN;
 //-- else{
 //--  res=new com.ibm.icu.math.BigDecimal(lint);
 //-- }
 //-- }
 //-- if (scale==0)
 //--  return res;
 //-- if (scale<0)
 //--  throw new java.lang.NumberFormatException("Negative scale:"+" "+scale);
 //-- res=clone(res); // safe copy [do not mutate]
 //-- res.exp=(int)-scale; // exponent is -scale
 //-- return res;
 //-- }

 /* ---------------------------------------------------------------- */
 /* Private methods                                                  */
 /* ---------------------------------------------------------------- */

 /* <sgml> Return char array value of a BigDecimal (conversion from
       BigDecimal to laid-out canonical char array).
    <p>The mantissa will either already have been rounded (following an
       operation) or will be of length appropriate (in the case of
       construction from an int, for example).
    <p>We must not alter the mantissa, here.
    <p>'form' describes whether we are to use exponential notation (and
       if so, which), or if we are to lay out as a plain/pure numeric.
    </sgml> */

 //--private char[] layout(){
 function layout() {
  //--char cmant[];
  var cmant;
  //--int i=0;
  var i=0;
  //--java.lang.StringBuffer sb=null;
  var sb=null;
  //--int euse=0;
  var euse=0;
  //--int sig=0;
  var sig=0;
  //--char csign=0;
  var csign=0;
  //--char rec[]=null;
  var rec=null;
  //--int needsign;
  var needsign;
  //--int mag;
  var mag;
  //--int len=0;
  var len=0;
  cmant=new Array(this.mant.length); // copy byte[] to a char[]
  {var $18=this.mant.length;i=0;i:for(;$18>0;$18--,i++){
   cmant[i]=this.mant[i]+'';
   }
  }/*i*/

  if (this.form!=MathContext.prototype.PLAIN)
   {/* exponential notation needed */
    //--sb=new java.lang.StringBuffer(cmant.length+15); // -x.xxxE+999999999
    sb="";
    if (this.ind==this.isneg)
     sb += '-';
    euse=(this.exp+cmant.length)-1; // exponent to use
    /* setup sig=significant digits and copy to result */
    if (this.form==MathContext.prototype.SCIENTIFIC)
     { // [default]
      sb += cmant[0]; // significant character
      if (cmant.length>1)  // have decimal part
       //--sb.append('.').append(cmant,1,cmant.length-1);
       sb += '.';
       sb += cmant.slice(1).join("");
     }
    else
     {engineering:do{
      sig=euse%3; // common
      if (sig<0)
       sig=3+sig; // negative exponent
      euse=euse-sig;
      sig++;
      if (sig>=cmant.length)
       { // zero padding may be needed
        //--sb.append(cmant,0,cmant.length);
        sb += cmant.join("");
        {var $19=sig-cmant.length;for(;$19>0;$19--){
         sb += '0';
         }
        }
       }
      else
       { // decimal point needed
        //--sb.append(cmant,0,sig).append('.').append(cmant,sig,cmant.length-sig);
        sb += cmant.slice(0,sig).join("");
        sb += '.';
        sb += cmant.slice(sig).join("");
       }
     }while(false);}/*engineering*/
    if (euse!=0)
     {
      if (euse<0)
       {
        csign='-';
        euse=-euse;
       }
      else
       csign='+';
      //--sb.append('E').append(csign).append(euse);
      sb += 'E';
      sb += csign;
      sb += euse;
     }
    //--rec=new Array(sb.length);
    //--Utility.getChars(sb, 0,sb.length(),rec,0);
    //--return rec;
    return sb.split("");
   }

  /* Here for non-exponential (plain) notation */
  if (this.exp==0)
   {/* easy */
    if (this.ind>=0)
     return cmant; // non-negative integer
    rec=new Array(cmant.length+1);
    rec[0]='-';
    //--java.lang.System.arraycopy((java.lang.Object)cmant,0,(java.lang.Object)rec,1,cmant.length);
    this.arraycopy(cmant,0,rec,1,cmant.length);
    return rec;
   }

  /* Need a '.' and/or some zeros */
  needsign=((this.ind==this.isneg)?1:0); // space for sign?  0 or 1

  /* MAG is the position of the point in the mantissa (index of the
     character it follows) */
  mag=this.exp+cmant.length;

  if (mag<1)
   {/* 0.00xxxx form */
    len=(needsign+2)-this.exp; // needsign+2+(-mag)+cmant.length
    rec=new Array(len);
    if (needsign!=0)
     rec[0]='-';
    rec[needsign]='0';
    rec[needsign+1]='.';
    {var $20=-mag;i=needsign+2;i:for(;$20>0;$20--,i++){ // maybe none
     rec[i]='0';
     }
    }/*i*/
    //--java.lang.System.arraycopy((java.lang.Object)cmant,0,(java.lang.Object)rec,(needsign+2)-mag,cmant.length);
    this.arraycopy(cmant,0,rec,(needsign+2)-mag,cmant.length);
    return rec;
   }

  if (mag>cmant.length)
   {/* xxxx0000 form */
    len=needsign+mag;
    rec=new Array(len);
    if (needsign!=0)
     rec[0]='-';
    //--java.lang.System.arraycopy((java.lang.Object)cmant,0,(java.lang.Object)rec,needsign,cmant.length);
    this.arraycopy(cmant,0,rec,needsign,cmant.length);
    {var $21=mag-cmant.length;i=needsign+cmant.length;i:for(;$21>0;$21--,i++){ // never 0
     rec[i]='0';
     }
    }/*i*/
    return rec;
   }

  /* decimal point is in the middle of the mantissa */
  len=(needsign+1)+cmant.length;
  rec=new Array(len);
  if (needsign!=0)
   rec[0]='-';
  //--java.lang.System.arraycopy((java.lang.Object)cmant,0,(java.lang.Object)rec,needsign,mag);
  this.arraycopy(cmant,0,rec,needsign,mag);
  rec[needsign+mag]='.';
  //--java.lang.System.arraycopy((java.lang.Object)cmant,mag,(java.lang.Object)rec,(needsign+mag)+1,cmant.length-mag);
  this.arraycopy(cmant,mag,rec,(needsign+mag)+1,cmant.length-mag);
  return rec;
  }

 /* <sgml> Checks a BigDecimal argument to ensure it's a true integer
       in a given range.
    <p>If OK, returns it as an int. </sgml> */
 // [currently only used by pow]

 //--private int intcheck(int min,int max){
 function intcheck(min, max) {
  //--int i;
  var i;
  i=this.intValueExact(); // [checks for non-0 decimal part]
  // Use same message as though intValueExact failed due to size
  if ((i<min)||(i>max))
   throw "intcheck(): Conversion overflow: "+i;
  return i;
  }

 /* <sgml> Carry out division operations. </sgml> */
 /*
    Arg1 is operation code: D=divide, I=integer divide, R=remainder
    Arg2 is the rhs.
    Arg3 is the context.
    Arg4 is explicit scale iff code='D' or 'I' (-1 if none).

    Underlying algorithm (complications for Remainder function and
    scaled division are omitted for clarity):

      Test for x/0 and then 0/x
      Exp =Exp1 - Exp2
      Exp =Exp +len(var1) -len(var2)
      Sign=Sign1 * Sign2
      Pad accumulator (Var1) to double-length with 0's (pad1)
      Pad Var2 to same length as Var1
      B2B=1st two digits of var2, +1 to allow for roundup
      have=0
      Do until (have=digits+1 OR residue=0)
        if exp<0 then if integer divide/residue then leave
        this_digit=0
        Do forever
           compare numbers
           if <0 then leave inner_loop
           if =0 then (- quick exit without subtract -) do
              this_digit=this_digit+1; output this_digit
              leave outer_loop; end
           Compare lengths of numbers (mantissae):
           If same then CA=first_digit_of_Var1
                   else CA=first_two_digits_of_Var1
           mult=ca*10/b2b   -- Good and safe guess at divisor
           if mult=0 then mult=1
           this_digit=this_digit+mult
           subtract
           end inner_loop
         if have\=0 | this_digit\=0 then do
           output this_digit
           have=have+1; end
         var2=var2/10
         exp=exp-1
         end outer_loop
      exp=exp+1   -- set the proper exponent
      if have=0 then generate answer=0
      Return to FINISHED
      Result defined by MATHV1

    For extended commentary, see DMSRCN.
  */

 //--private com.ibm.icu.math.BigDecimal dodivide(char code,com.ibm.icu.math.BigDecimal rhs,com.ibm.icu.math.MathContext set,int scale){
 function dodivide(code, rhs, set, scale) {
  //--com.ibm.icu.math.BigDecimal lhs;
  var lhs;
  //--int reqdig;
  var reqdig;
  //--int newexp;
  var newexp;
  //--com.ibm.icu.math.BigDecimal res;
  var res;
  //--int newlen;
  var newlen;
  //--byte var1[];
  var var1;
  //--int var1len;
  var var1len;
  //--byte var2[];
  var var2;
  //--int var2len;
  var var2len;
  //--int b2b;
  var b2b;
  //--int have;
  var have;
  //--int thisdigit=0;
  var thisdigit=0;
  //--int i=0;
  var i=0;
  //--byte v2=0;
  var v2=0;
  //--int ba=0;
  var ba=0;
  //--int mult=0;
  var mult=0;
  //--int start=0;
  var start=0;
  //--int padding=0;
  var padding=0;
  //--int d=0;
  var d=0;
  //--byte newvar1[]=null;
  var newvar1=null;
  //--byte lasthave=0;
  var lasthave=0;
  //--int actdig=0;
  var actdig=0;
  //--byte newmant[]=null;
  var newmant=null;

  if (set.lostDigits)
   this.checkdigits(rhs,set.digits);
  lhs=this; // name for clarity

  // [note we must have checked lostDigits before the following checks]
  if (rhs.ind==0)
   throw "dodivide(): Divide by 0"; // includes 0/0
  if (lhs.ind==0)
   { // 0/x => 0 [possibly with .0s]
    if (set.form!=MathContext.prototype.PLAIN)
     return this.ZERO;
    if (scale==(-1))
     return lhs;
    return lhs.setScale(scale);
   }

  /* Prepare numbers according to BigDecimal rules */
  reqdig=set.digits; // local copy (heavily used)
  if (reqdig>0)
   {
    if (lhs.mant.length>reqdig)
     lhs=this.clone(lhs).round(set);
    if (rhs.mant.length>reqdig)
     rhs=this.clone(rhs).round(set);
   }
  else
   {/* scaled divide */
    if (scale==(-1))
     scale=lhs.scale();
    // set reqdig to be at least large enough for the computation
    reqdig=lhs.mant.length; // base length
    // next line handles both positive lhs.exp and also scale mismatch
    if (scale!=(-lhs.exp))
     reqdig=(reqdig+scale)+lhs.exp;
    reqdig=(reqdig-((rhs.mant.length-1)))-rhs.exp; // reduce by RHS effect
    if (reqdig<lhs.mant.length)
     reqdig=lhs.mant.length; // clamp
    if (reqdig<rhs.mant.length)
     reqdig=rhs.mant.length; // ..
   }

  /* precalculate exponent */
  newexp=((lhs.exp-rhs.exp)+lhs.mant.length)-rhs.mant.length;
  /* If new exponent -ve, then some quick exits are possible */
  if (newexp<0)
   if (code!='D')
    {
     if (code=='I')
      return this.ZERO; // easy - no integer part
     /* Must be 'R'; remainder is [finished clone of] input value */
     return this.clone(lhs).finish(set,false);
    }

  /* We need slow division */
  res=new BigDecimal(); // where we'll build result
  res.ind=(lhs.ind*rhs.ind); // final sign (for D/I)
  res.exp=newexp; // initial exponent (for D/I)
  res.mant=this.createArrayWithZeros(reqdig+1); // where build the result

  /* Now [virtually pad the mantissae with trailing zeros */
  // Also copy the LHS, which will be our working array
  newlen=(reqdig+reqdig)+1;
  var1=this.extend(lhs.mant,newlen); // always makes longer, so new safe array
  var1len=newlen; // [remaining digits are 0]

  var2=rhs.mant;
  var2len=newlen;

  /* Calculate first two digits of rhs (var2), +1 for later estimations */
  b2b=(var2[0]*10)+1;
  if (var2.length>1)
   b2b=b2b+var2[1];

  /* start the long-division loops */
  have=0;
  {outer:for(;;){
   thisdigit=0;
   /* find the next digit */
   {inner:for(;;){
    if (var1len<var2len)
     break inner; // V1 too low
    if (var1len==var2len)
     { // compare needed
      {compare:do{ // comparison
       {var $22=var1len;i=0;i:for(;$22>0;$22--,i++){
        // var1len is always <= var1.length
        if (i<var2.length)
         v2=var2[i];
        else
         v2=0;
        if (var1[i]<v2)
         break inner; // V1 too low
        if (var1[i]>v2)
         break compare; // OK to subtract
        }
       }/*i*/
       /* reach here if lhs and rhs are identical; subtraction will
          increase digit by one, and the residue will be 0 so we
          are done; leave the loop with residue set to 0 (in case
          code is 'R' or ROUND_UNNECESSARY or a ROUND_HALF_xxxx is
          being checked) */
       thisdigit++;
       res.mant[have]=thisdigit;
       have++;
       var1[0]=0; // residue to 0 [this is all we'll test]
       // var1len=1      -- [optimized out]
       break outer;
      }while(false);}/*compare*/
      /* prepare for subtraction.  Estimate BA (lengths the same) */
      ba=var1[0]; // use only first digit
     } // lengths the same
    else
     {/* lhs longer than rhs */
      /* use first two digits for estimate */
      ba=var1[0]*10;
      if (var1len>1)
       ba=ba+var1[1];
     }
    /* subtraction needed; V1>=V2 */
    mult=div((ba*10),b2b);
    if (mult==0)
     mult=1;
    thisdigit=thisdigit+mult;
    // subtract; var1 reusable
    var1=this.byteaddsub(var1,var1len,var2,var2len,-mult,true);
    if (var1[0]!=0)
     continue inner; // maybe another subtract needed
    /* V1 now probably has leading zeros, remove leading 0's and try
       again. (It could be longer than V2) */
    {var $23=var1len-2;start=0;start:for(;start<=$23;start++){
     if (var1[start]!=0)
      break start;
     var1len--;
     }
    }/*start*/
    if (start==0)
     continue inner;
    // shift left
    //--java.lang.System.arraycopy((java.lang.Object)var1,start,(java.lang.Object)var1,0,var1len);
    this.arraycopy(var1,start,var1,0,var1len);
    }
   }/*inner*/

   /* We have the next digit */
   if ((have!=0)||(thisdigit!=0))
    { // put the digit we got
     res.mant[have]=thisdigit;
     have++;
     if (have==(reqdig+1))
      break outer; // we have all we need
     if (var1[0]==0)
      break outer; // residue now 0
    }
   /* can leave now if a scaled divide and exponent is small enough */
   if (scale>=0)
    if ((-res.exp)>scale)
     break outer;
   /* can leave now if not Divide and no integer part left  */
   if (code!='D')
    if (res.exp<=0)
     break outer;
   res.exp=res.exp-1; // reduce the exponent
   /* to get here, V1 is less than V2, so divide V2 by 10 and go for
      the next digit */
   var2len--;
   }
  }/*outer*/

  /* here when we have finished dividing, for some reason */
  // have is the number of digits we collected in res.mant
  if (have==0)
   have=1; // res.mant[0] is 0; we always want a digit

  if ((code=='I')||(code=='R'))
   {/* check for integer overflow needed */
    if ((have+res.exp)>reqdig)
     throw "dodivide(): Integer overflow";

    if (code=='R')
     {remainder:do{
      /* We were doing Remainder -- return the residue */
      if (res.mant[0]==0)  // no integer part was found
       return this.clone(lhs).finish(set,false); // .. so return lhs, canonical
      if (var1[0]==0)
       return this.ZERO; // simple 0 residue
      res.ind=lhs.ind; // sign is always as LHS
      /* Calculate the exponent by subtracting the number of padding zeros
         we added and adding the original exponent */
      padding=((reqdig+reqdig)+1)-lhs.mant.length;
      res.exp=(res.exp-padding)+lhs.exp;

      /* strip insignificant padding zeros from residue, and create/copy
         the resulting mantissa if need be */
      d=var1len;
      {i=d-1;i:for(;i>=1;i--){if(!((res.exp<lhs.exp)&&(res.exp<rhs.exp)))break;
       if (var1[i]!=0)
        break i;
       d--;
       res.exp=res.exp+1;
       }
      }/*i*/
      if (d<var1.length)
       {/* need to reduce */
        newvar1=new Array(d);
        //--java.lang.System.arraycopy((java.lang.Object)var1,0,(java.lang.Object)newvar1,0,d); // shorten
        this.arraycopy(var1,0,newvar1,0,d);
        var1=newvar1;
       }
      res.mant=var1;
      return res.finish(set,false);
     }while(false);}/*remainder*/
   }

  else
   {/* 'D' -- no overflow check needed */
    // If there was a residue then bump the final digit (iff 0 or 5)
    // so that the residue is visible for ROUND_UP, ROUND_HALF_xxx and
    // ROUND_UNNECESSARY checks (etc.) later.
    // [if we finished early, the residue will be 0]
    if (var1[0]!=0)
     { // residue not 0
      lasthave=res.mant[have-1];
      if (((lasthave%5))==0)
       res.mant[have-1]=(lasthave+1);
     }
   }

  /* Here for Divide or Integer Divide */
  // handle scaled results first ['I' always scale 0, optional for 'D']
  if (scale>=0)
   {scaled:do{
    // say 'scale have res.exp len' scale have res.exp res.mant.length
    if (have!=res.mant.length)
     // already padded with 0's, so just adjust exponent
     res.exp=res.exp-((res.mant.length-have));
    // calculate number of digits we really want [may be 0]
    actdig=res.mant.length-(((-res.exp)-scale));
    res.round(actdig,set.roundingMode); // round to desired length
    // This could have shifted left if round (say) 0.9->1[.0]
    // Repair if so by adding a zero and reducing exponent
    if (res.exp!=(-scale))
     {
      res.mant=this.extend(res.mant,res.mant.length+1);
      res.exp=res.exp-1;
     }
    return res.finish(set,true); // [strip if not PLAIN]
   }while(false);}/*scaled*/

  // reach here only if a non-scaled
  if (have==res.mant.length)
   { // got digits+1 digits
    res.round(set);
    have=reqdig;
   }
  else
   {/* have<=reqdig */
    if (res.mant[0]==0)
     return this.ZERO; // fastpath
    // make the mantissa truly just 'have' long
    // [we could let finish do this, during strip, if we adjusted
    // the exponent; however, truncation avoids the strip loop]
    newmant=new Array(have); // shorten
    //--java.lang.System.arraycopy((java.lang.Object)res.mant,0,(java.lang.Object)newmant,0,have);
    this.arraycopy(res.mant,0,newmant,0,have);
    res.mant=newmant;
   }
  return res.finish(set,true);
  }

 /* <sgml> Report a conversion exception. </sgml> */

 //--private void bad(char s[]){
 function bad(prefix, s) {
  throw prefix + "Not a number: "+s;
  }

 /* <sgml> Report a bad argument to a method. </sgml>
    Arg1 is method name
    Arg2 is argument position
    Arg3 is what was found */

 //--private void badarg(java.lang.String name,int pos,java.lang.String value){
 function badarg(name, pos, value) {
  throw "Bad argument "+pos+" to "+name+": "+value;
  }

 /* <sgml> Extend byte array to given length, padding with 0s.  If no
    extension is required then return the same array. </sgml>

    Arg1 is the source byte array
    Arg2 is the new length (longer)
    */

 //--private static final byte[] extend(byte inarr[],int newlen){
 function extend(inarr, newlen) {
  //--byte newarr[];
  var newarr;
  if (inarr.length==newlen)
   return inarr;
  newarr=createArrayWithZeros(newlen);
  //--java.lang.System.arraycopy((java.lang.Object)inarr,0,(java.lang.Object)newarr,0,inarr.length);
  this.arraycopy(inarr,0,newarr,0,inarr.length);
  // 0 padding is carried out by the JVM on allocation initialization
  return newarr;
  }

 /* <sgml> Add or subtract two >=0 integers in byte arrays
    <p>This routine performs the calculation:
    <pre>
    C=A+(B*M)
    </pre>
    Where M is in the range -9 through +9
    <p>
    If M<0 then A>=B must be true, so the result is always
    non-negative.

    Leading zeros are not removed after a subtraction.  The result is
    either the same length as the longer of A and B, or 1 longer than
    that (if a carry occurred).

    A is not altered unless Arg6 is 1.
    B is never altered.

    Arg1 is A
    Arg2 is A length to use (if longer than A, pad with 0's)
    Arg3 is B
    Arg4 is B length to use (if longer than B, pad with 0's)
    Arg5 is M, the multiplier
    Arg6 is 1 if A can be used to build the result (if it fits)

    This routine is severely performance-critical; *any* change here
    must be measured (timed) to assure no performance degradation.
    */
 // 1996.02.20 -- enhanced version of DMSRCN algorithm (1981)
 // 1997.10.05 -- changed to byte arrays (from char arrays)
 // 1998.07.01 -- changed to allow destructive reuse of LHS
 // 1998.07.01 -- changed to allow virtual lengths for the arrays
 // 1998.12.29 -- use lookaside for digit/carry calculation
 // 1999.08.07 -- avoid multiply when mult=1, and make db an int
 // 1999.12.22 -- special case m=-1, also drop 0 special case

 //--private static final byte[] byteaddsub(byte a[],int avlen,byte b[],int bvlen,int m,boolean reuse){
 function byteaddsub(a, avlen, b, bvlen, m, reuse) {
  //--int alength;
  var alength;
  //--int blength;
  var blength;
  //--int ap;
  var ap;
  //--int bp;
  var bp;
  //--int maxarr;
  var maxarr;
  //--byte reb[];
  var reb;
  //--boolean quickm;
  var quickm;
  //--int digit;
  var digit;
  //--int op=0;
  var op=0;
  //--int dp90=0;
  var dp90=0;
  //--byte newarr[];
  var newarr;
  //--int i=0;
  var i=0;




  // We'll usually be right if we assume no carry
  alength=a.length; // physical lengths
  blength=b.length; // ..
  ap=avlen-1; // -> final (rightmost) digit
  bp=bvlen-1; // ..
  maxarr=bp;
  if (maxarr<ap)
   maxarr=ap;
  reb=null; // result byte array
  if (reuse)
   if ((maxarr+1)==alength)
    reb=a; // OK to reuse A
  if (reb==null){
   reb=this.createArrayWithZeros(maxarr+1); // need new array
   }

  quickm=false; // 1 if no multiply needed
  if (m==1)
   quickm=true; // most common
  else
   if (m==(-1))
    quickm=true; // also common

  digit=0; // digit, with carry or borrow
  {op=maxarr;op:for(;op>=0;op--){
   if (ap>=0)
    {
     if (ap<alength)
      digit=digit+a[ap]; // within A
     ap--;
    }
   if (bp>=0)
    {
     if (bp<blength)
      { // within B
       if (quickm)
        {
         if (m>0)
          digit=digit+b[bp]; // most common
         else
          digit=digit-b[bp]; // also common
        }
       else
        digit=digit+(b[bp]*m);
      }
     bp--;
    }
   /* result so far (digit) could be -90 through 99 */
   if (digit<10)
    if (digit>=0)
     {quick:do{ // 0-9
      reb[op]=digit;
      digit=0; // no carry
      continue op;
     }while(false);}/*quick*/
   dp90=digit+90;
   reb[op]=this.bytedig[dp90]; // this digit
   digit=this.bytecar[dp90]; // carry or borrow
   }
  }/*op*/

  if (digit==0)
   return reb; // no carry
  // following line will become an Assert, later
  // if digit<0 then signal ArithmeticException("internal.error ["digit"]")

  /* We have carry -- need to make space for the extra digit */
  newarr=null;
  if (reuse)
   if ((maxarr+2)==a.length)
    newarr=a; // OK to reuse A
  if (newarr==null)
   newarr=new Array(maxarr+2);
  newarr[0]=digit; // the carried digit ..
  // .. and all the rest [use local loop for short numbers]
  //--if (maxarr<10)
   {var $24=maxarr+1;i=0;i:for(;$24>0;$24--,i++){
    newarr[i+1]=reb[i];
    }
   }/*i*/
  //--else
   //--java.lang.System.arraycopy((java.lang.Object)reb,0,(java.lang.Object)newarr,1,maxarr+1);
  return newarr;
  }

 /* <sgml> Initializer for digit array properties (lookaside). </sgml>
    Returns the digit array, and initializes the carry array. */

 //--private static final byte[] diginit(){
 function diginit() {
  //--byte work[];
  var work;
  //--int op=0;
  var op=0;
  //--int digit=0;
  var digit=0;
  work=new Array((90+99)+1);
  {op=0;op:for(;op<=(90+99);op++){
   digit=op-90;
   if (digit>=0)
    {
     work[op]=(digit%10);
     BigDecimal.prototype.bytecar[op]=(div(digit,10)); // calculate carry
     continue op;
    }
   // borrowing...
   digit=digit+100; // yes, this is right [consider -50]
   work[op]=(digit%10);
   BigDecimal.prototype.bytecar[op]=((div(digit,10))-10); // calculate borrow [NB: - after %]
   }
  }/*op*/
  return work;
  }

 /* <sgml> Create a copy of BigDecimal object for local use.
    <p>This does NOT make a copy of the mantissa array.
    </sgml>
    Arg1 is the BigDecimal to clone (non-null)
    */

 //--private static final com.ibm.icu.math.BigDecimal clone(com.ibm.icu.math.BigDecimal dec){
 function clone(dec) {
  //--com.ibm.icu.math.BigDecimal copy;
  var copy;
  copy=new BigDecimal();
  copy.ind=dec.ind;
  copy.exp=dec.exp;
  copy.form=dec.form;
  copy.mant=dec.mant;
  return copy;
  }

 /* <sgml> Check one or two numbers for lost digits. </sgml>
    Arg1 is RHS (or null, if none)
    Arg2 is current DIGITS setting
    returns quietly or throws an exception */

 //--private void checkdigits(com.ibm.icu.math.BigDecimal rhs,int dig){
 function checkdigits(rhs, dig) {
  if (dig==0)
   return; // don't check if digits=0
  // first check lhs...
  if (this.mant.length>dig)
   if ((!(this.allzero(this.mant,dig))))
    throw "Too many digits: "+this.toString();
  if (rhs==null)
   return; // monadic
  if (rhs.mant.length>dig)
   if ((!(this.allzero(rhs.mant,dig))))
    throw "Too many digits: "+rhs.toString();
  return;
  }

 /* <sgml> Round to specified digits, if necessary. </sgml>
    Arg1 is requested MathContext [with length and rounding mode]
    returns this, for convenience */

 //--private com.ibm.icu.math.BigDecimal round(com.ibm.icu.math.MathContext set){
 //-- return round(set.digits,set.roundingMode);
 //-- }

 /* <sgml> Round to specified digits, if necessary.
    Arg1 is requested length (digits to round to)
            [may be <=0 when called from format, dodivide, etc.]
    Arg2 is rounding mode
    returns this, for convenience

    ind and exp are adjusted, but not cleared for a mantissa of zero

    The length of the mantissa returned will be Arg1, except when Arg1
    is 0, in which case the returned mantissa length will be 1.
    </sgml>
    */

 //private com.ibm.icu.math.BigDecimal round(int len,int mode){
 function round() {
  var len;
  var mode;
  if (round.arguments.length == 2)
   {
    len = round.arguments[0];
    mode = round.arguments[1];
   }
  else if (round.arguments.length == 1)
   {
    var set = round.arguments[0];
    len = set.digits;
    mode = set.roundingMode;
   }
  else
   {
    throw "round(): " + round.arguments.length + " arguments given; expected 1 or 2";
   }
  //int adjust;
  var adjust;
  //int sign;
  var sign;
  //byte oldmant[];
  var oldmant;
  //boolean reuse=false;
  var reuse=false;
  //--byte first=0;
  var first=0;
  //--int increment;
  var increment;
  //--byte newmant[]=null;
  var newmant=null;
  adjust=this.mant.length-len;
  if (adjust<=0)
   return this; // nowt to do

  this.exp=this.exp+adjust; // exponent of result
  sign=this.ind; // save [assumes -1, 0, 1]
  oldmant=this.mant; // save
  if (len>0)
   {
    // remove the unwanted digits
    this.mant=new Array(len);
    //--java.lang.System.arraycopy((java.lang.Object)oldmant,0,(java.lang.Object)mant,0,len);
    this.arraycopy(oldmant,0,this.mant,0,len);
    reuse=true; // can reuse mantissa
    first=oldmant[len]; // first of discarded digits
   }
  else
   {/* len<=0 */
    this.mant=this.ZERO.mant;
    this.ind=this.iszero;
    reuse=false; // cannot reuse mantissa
    if (len==0)
     first=oldmant[0];
    else
     first=0; // [virtual digit]
   }

  // decide rounding adjustment depending on mode, sign, and discarded digits
  increment=0; // bumper
  {modes:do{/*select*/
  if (mode==this.ROUND_HALF_UP)
   { // default first [most common]
    if (first>=5)
     increment=sign;
   }
  else if (mode==this.ROUND_UNNECESSARY)
   { // default for setScale()
    // discarding any non-zero digits is an error
    if ((!(this.allzero(oldmant,len))))
     throw "round(): Rounding necessary";
   }
  else if (mode==this.ROUND_HALF_DOWN)
   { // 0.5000 goes down
    if (first>5)
     increment=sign;
    else
     if (first==5)
      if ((!(this.allzero(oldmant,len+1))))
       increment=sign;
   }
  else if (mode==this.ROUND_HALF_EVEN)
   { // 0.5000 goes down if left digit even
    if (first>5)
     increment=sign;
    else
     if (first==5)
      {
       if ((!(this.allzero(oldmant,len+1))))
        increment=sign;
       else /* 0.5000 */
        if ((((this.mant[this.mant.length-1])%2))==1)
         increment=sign;
      }
   }
  else if (mode==this.ROUND_DOWN)
   {} // never increment
  else if (mode==this.ROUND_UP)
   { // increment if discarded non-zero
    if ((!(this.allzero(oldmant,len))))
     increment=sign;
   }
  else if (mode==this.ROUND_CEILING)
   { // more positive
    if (sign>0)
     if ((!(this.allzero(oldmant,len))))
      increment=sign;
   }
  else if (mode==this.ROUND_FLOOR)
   { // more negative
    if (sign<0)
     if ((!(this.allzero(oldmant,len))))
      increment=sign;
   }
  else{
   throw "round(): Bad round value: "+mode;
  }
  }while(false);}/*modes*/

  if (increment!=0)
   {bump:do{
    if (this.ind==this.iszero)
     {
      // we must not subtract from 0, but result is trivial anyway
      this.mant=this.ONE.mant;
      this.ind=increment;
     }
    else
     {
      // mantissa is non-0; we can safely add or subtract 1
      if (this.ind==this.isneg)
       increment=-increment;
      newmant=this.byteaddsub(this.mant,this.mant.length,this.ONE.mant,1,increment,reuse);
      if (newmant.length>this.mant.length)
       { // had a carry
        // drop rightmost digit and raise exponent
        this.exp++;
        // mant is already the correct length
        //java.lang.System.arraycopy((java.lang.Object)newmant,0,(java.lang.Object)mant,0,mant.length);
        this.arraycopy(newmant,0,this.mant,0,this.mant.length);
       }
      else
       this.mant=newmant;
     }
   }while(false);}/*bump*/
  // rounding can increase exponent significantly
  if (this.exp>this.MaxExp)
   throw "round(): Exponent Overflow: "+this.exp;
  return this;
  }

 /* <sgml> Test if rightmost digits are all 0.
    Arg1 is a mantissa array to test
    Arg2 is the offset of first digit to check
            [may be negative; if so, digits to left are 0's]
    returns 1 if all the digits starting at Arg2 are 0

    Arg2 may be beyond array bounds, in which case 1 is returned
    </sgml> */

 //--private static final boolean allzero(byte array[],int start){
 function allzero(array, start) {
  //--int i=0;
  var i=0;
  if (start<0)
   start=0;
  {var $25=array.length-1;i=start;i:for(;i<=$25;i++){
   if (array[i]!=0)
    return false;
   }
  }/*i*/
  return true;
  }

 /* <sgml> Carry out final checks and canonicalization
    <p>
    This finishes off the current number by:
      1. Rounding if necessary (NB: length includes leading zeros)
      2. Stripping trailing zeros (if requested and \PLAIN)
      3. Stripping leading zeros (always)
      4. Selecting exponential notation (if required)
      5. Converting a zero result to just '0' (if \PLAIN)
    In practice, these operations overlap and share code.
    It always sets form.
    </sgml>
    Arg1 is requested MathContext (length to round to, trigger, and FORM)
    Arg2 is 1 if trailing insignificant zeros should be removed after
         round (for division, etc.), provided that set.form isn't PLAIN.
   returns this, for convenience
   */

 //--private com.ibm.icu.math.BigDecimal finish(com.ibm.icu.math.MathContext set,boolean strip){
 function finish(set, strip) {
  //--int d=0;
  var d=0;
  //--int i=0;
  var i=0;
  //--byte newmant[]=null;
  var newmant=null;
  //--int mag=0;
  var mag=0;
  //--int sig=0;
  var sig=0;
  /* Round if mantissa too long and digits requested */
  if (set.digits!=0)
   if (this.mant.length>set.digits)
    this.round(set);

  /* If strip requested (and standard formatting), remove
     insignificant trailing zeros. */
  if (strip)
   if (set.form!=MathContext.prototype.PLAIN)
    {
     d=this.mant.length;
     /* see if we need to drop any trailing zeros */
     {i=d-1;i:for(;i>=1;i--){
      if (this.mant[i]!=0)
       break i;
      d--;
      this.exp++;
      }
     }/*i*/
     if (d<this.mant.length)
      {/* need to reduce */
       newmant=new Array(d);
       //--java.lang.System.arraycopy((java.lang.Object)this.mant,0,(java.lang.Object)newmant,0,d);
       this.arraycopy(this.mant,0,newmant,0,d);
       this.mant=newmant;
      }
    }

  this.form=MathContext.prototype.PLAIN; // preset

  /* Now check for leading- and all- zeros in mantissa */
  {var $26=this.mant.length;i=0;i:for(;$26>0;$26--,i++){
   if (this.mant[i]!=0)
    {
     // non-0 result; ind will be correct
     // remove leading zeros [e.g., after subtract]
     if (i>0)
      {delead:do{
       newmant=new Array(this.mant.length-i);
       //--java.lang.System.arraycopy((java.lang.Object)this.mant,i,(java.lang.Object)newmant,0,this.mant.length-i);
       this.arraycopy(this.mant,i,newmant,0,this.mant.length-i);
       this.mant=newmant;
      }while(false);}/*delead*/
     // now determine form if not PLAIN
     mag=this.exp+this.mant.length;
     if (mag>0)
      { // most common path
       if (mag>set.digits)
        if (set.digits!=0)
         this.form=set.form;
       if ((mag-1)<=this.MaxExp)
        return this; // no overflow; quick return
      }
     else
      if (mag<(-5))
       this.form=set.form;
     /* check for overflow */
     mag--;
     if ((mag<this.MinExp)||(mag>this.MaxExp))
      {overflow:do{
       // possible reprieve if form is engineering
       if (this.form==MathContext.prototype.ENGINEERING)
        {
         sig=mag%3; // leftover
         if (sig<0)
          sig=3+sig; // negative exponent
         mag=mag-sig; // exponent to use
         // 1999.06.29: second test here must be MaxExp
         if (mag>=this.MinExp)
          if (mag<=this.MaxExp)
           break overflow;
        }
       throw "finish(): Exponent Overflow: "+mag;
      }while(false);}/*overflow*/
     return this;
    }
   }
  }/*i*/

  // Drop through to here only if mantissa is all zeros
  this.ind=this.iszero;
  {/*select*/
  if (set.form!=MathContext.prototype.PLAIN)
   this.exp=0; // standard result; go to '0'
  else if (this.exp>0)
   this.exp=0; // +ve exponent also goes to '0'
  else{
   // a plain number with -ve exponent; preserve and check exponent
   if (this.exp<this.MinExp)
    throw "finish(): Exponent Overflow: "+this.exp;
  }
  }
  this.mant=this.ZERO.mant; // canonical mantissa
  return this;
  }

 function isGreaterThan(other) {
  return this.compareTo(other) > 0;
 };
 function isLessThan(other) {
  return this.compareTo(other) < 0;
 };
 function isGreaterThanOrEqualTo(other) {
  return this.compareTo(other) >= 0;
 };
 function isLessThanOrEqualTo(other) {
  return this.compareTo(other) <= 0;
 };
 function isPositive() {
  return this.compareTo(BigDecimal.prototype.ZERO) > 0;
 };
 function isNegative() {
  return this.compareTo(BigDecimal.prototype.ZERO) < 0;
 };
 function isZero() {
  return this.compareTo(BigDecimal.prototype.ZERO) === 0;
 };
return BigDecimal;
})(MathContext); // BigDecimal depends on MathContext

if (typeof define === "function" && define.amd != null) {
	// AMD-loader compatible resource declaration
	// require('bigdecimal') will return JS Object:
	// {'BigDecimal':BigDecimalPointer, 'MathContext':MathContextPointer}
	define({'BigDecimal':BigDecimal, 'MathContext':MathContext});
} else if (typeof this === "object"){
	// global-polluting outcome.
	this.BigDecimal = BigDecimal;
	this.MathContext = MathContext;
}

}).call(this); // in browser 'this' will be 'window' or simulated window object in AMD-loading scenarios.
