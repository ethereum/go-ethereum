#include <libserpent/funcs.h>
#include <libserpent/bignum.h>
#include <iostream>

using namespace std;

int main() {
	cout << printAST(compileToLLL(get_file_contents("examples/namecoin.se"))) << "\n";
    cout << decimalSub("10234", "10234") << "\n";
    cout << decimalSub("10234", "10233") << "\n";
}
