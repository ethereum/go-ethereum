#include <stdio.h>
#include <iostream>
#include <vector>
#include <map>
#include "bignum.h"

//Integer to string conversion
std::string unsignedToDecimal(unsigned branch) {
    if (branch < 10) return nums.substr(branch, 1);
    else return unsignedToDecimal(branch / 10) + nums.substr(branch % 10,1);
}

//Add two strings representing decimal values
std::string decimalAdd(std::string a, std::string b) {
    std::string o = a;
    while (b.length() < a.length()) b = "0" + b;
    while (o.length() < b.length()) o = "0" + o;
    bool carry = false;
    for (int i = o.length() - 1; i >= 0; i--) {
        o[i] = o[i] + b[i] - '0';
        if (carry) o[i]++;
        if (o[i] > '9') {
            o[i] -= 10;
            carry = true;
        }
        else carry = false;
    }
    if (carry) o = "1" + o;
    return o;
}

//Helper function for decimalMul
std::string decimalDigitMul(std::string a, int dig) {
    if (dig == 0) return "0";
    else return decimalAdd(a, decimalDigitMul(a, dig - 1));
}

//Multiply two strings representing decimal values
std::string decimalMul(std::string a, std::string b) {
    std::string o = "0";
	for (unsigned i = 0; i < b.length(); i++) {
        std::string n = decimalDigitMul(a, b[i] - '0');
        if (n != "0") {
			for (unsigned j = i + 1; j < b.length(); j++) n += "0";
        }
        o = decimalAdd(o, n);
    }
    return o;
}

//Modexp
std::string decimalModExp(std::string b, std::string e, std::string m) {
    if (e == "0") return "1";
    else if (e == "1") return b;
    else if (decimalMod(e, "2") == "0") {
        std::string o = decimalModExp(b, decimalDiv(e, "2"), m);
        return decimalMod(decimalMul(o, o), m);
    }
    else {
        std::string o = decimalModExp(b, decimalDiv(e, "2"), m);
        return decimalMod(decimalMul(decimalMul(o, o), b), m);
    }
}

//Is a greater than b? Flag allows equality
bool decimalGt(std::string a, std::string b, bool eqAllowed) {
    if (a == b) return eqAllowed;
    return (a.length() > b.length()) || (a.length() >= b.length() && a > b);
}

//Subtract the two strings representing decimal values
std::string decimalSub(std::string a, std::string b) {
    if (b == "0") return a;
    if (b == a) return "0";
    while (b.length() < a.length()) b = "0" + b;
    std::string c = b;
	for (unsigned i = 0; i < c.length(); i++) c[i] = '0' + ('9' - c[i]);
    std::string o = decimalAdd(decimalAdd(a, c).substr(1), "1");
    while (o.size() > 1 && o[0] == '0') o = o.substr(1);
    return o;
}

//Divide the two strings representing decimal values
std::string decimalDiv(std::string a, std::string b) {
    std::string c = b;
    if (decimalGt(c, a)) return "0";
    int zeroes = -1;
    while (decimalGt(a, c, true)) {
        zeroes += 1;
        c = c + "0";
    }
    c = c.substr(0, c.size() - 1);
    std::string quot = "0";
    while (decimalGt(a, c, true)) {
        a = decimalSub(a, c);
        quot = decimalAdd(quot, "1");
    }
    for (int i = 0; i < zeroes; i++) quot += "0";
    return decimalAdd(quot, decimalDiv(a, b));
}

//Modulo the two strings representing decimal values
std::string decimalMod(std::string a, std::string b) {
    return decimalSub(a, decimalMul(decimalDiv(a, b), b));
}

//String to int conversion
unsigned decimalToUnsigned(std::string a) {
    if (a.size() == 0) return 0;
    else return (a[a.size() - 1] - '0') 
        + decimalToUnsigned(a.substr(0,a.size()-1)) * 10;
}
