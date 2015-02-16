//
// To compile:
//
//   ragel -Z -G2 -o parseqt.go parseqt.rl
//
// To show a diagram of the state machine:
//
//   ragel -V -G2 -p -o parseqt.dot parseqt.rl
//   dot -Tsvg -o parseqt.svg parseqt.dot
//   chrome parseqt.svg
//

package main

import (
	"fmt"
)

%%{
	machine parseqt;

	write data;
}%%

func parseQt(data string, header *Header) error {
	var cs, p, pe int
	var ts, te, act, eof int

	pe = len(data)
	eof = len(data)

	_, _, _ = ts, te, act

	//stack := make([]int, 32)
	//top := 0

	var curline = 1

	var m0, m1, m2, m3, m4, m5, m6 int
	var foundclass int
	var inpublic bool
	var heading string
	var lineblock int
	var f Func

	%%{
		nl = '\n' @{ curline++ };
		cd = [^\n];
		sp = [ \t];
		id = [A-Za-z0-9_]+;
		spnl = ( sp | nl );

		main := |*
			'class Q_GUI_EXPORT ' id >{ m0 = p } %{ m1 = p } ' : public QAbstractOpenGLFunctions' nl '{' nl
			{
				header.Class = data[m0:m1]
				foundclass++
				fgoto inclass;
			};

			'class Q_GUI_EXPORT ' id >{ m0 = p } %{ m1 = p } nl '{' nl
			{
				if data[m0:m1] == "QOpenGLFunctions" {
					header.Class = data[m0:m1]
					foundclass++
				}
				fgoto inclass;
			};

			# Ignore any other line.
			cd* nl;
		*|;

		inclass := |*
			# Track heading comments.
			sp* '//' sp* cd* >{ m0 = p } @{ m1 = p } sp* nl
			{
				heading = data[m0:m1]
				_ = heading
				lineblock++
			};

			# Ignore constructor/destructor.
			sp* '~'? id >{ m0 = p } %{ m1 = p } '()' sp* ( ';' | '{}' ) sp* nl {
				if data[m0:m1] != header.Class {
					fbreak;
				}
			};

			# Ignore initialization function.
			sp* 'bool' sp+ 'initializeOpenGLFunctions()' cd* nl;

			# Ignore friend classes.
			sp* 'friend' sp+ 'class' sp+ id sp* ';' sp* nl;

			# Track public/private to ignore whatever isn't public.
			sp* 'public:' sp* nl
			{
				inpublic = true
			};
			sp* ( 'private:' | 'protected:' ) sp* nl
			{
				inpublic = false
			};

			# Record function prototypes.
			sp* ( 'const' sp+ )? id >{ m0 = p } %{ m1 = p; m4 = 0 } ( sp 'const'? | '*'+ ${ m4++ } )+
				# Name
				'gl' >{ m2 = p } id %{ m3 = p; f = Func{Name: data[m2:m3], Type: data[m0:m1], Addr: m4} } sp* '(' >{ m6 = 0 } sp*
				# Parameters
				( 'void'? sp* ')' | ( ( 'const' %{ m6 = 1 } sp+ )? id >{ m0 = p } %{ m1 = p; m4 = 0 } ( sp 'const'? | '*' ${ m4++ } )+ id >{ m2 = p; m5 = 0 } %{ m3 = p } ( '[' [0-9]+ ${ m5 = m5*10 + (int(data[p]) - '0') } ']' )? sp* [,)]
					>{ f.Param = append(f.Param, Param{Name: data[m2:m3], Type: data[m0:m1], Addr: m4, Array: m5, Const: m6 > 0}); m6 = 0 } sp* )+ )
				sp* ';'
			{
				if (inpublic) {
					header.Func = append(header.Func, f)
				}
			};

			# Record feature flags.
			sp* 'enum OpenGLFeature' sp* nl sp* '{' sp* nl
				( sp* id >{ m0 = p } %{ m1 = p } sp* '=' sp* '0x' >{ m2 = p } [0-9]+ %{ m3 = p } ','? sp* nl
					>{ header.FeatureFlags = append(header.FeatureFlags, Const{Name: data[m0:m1], Value: data[m2:m3]}) } )+
			sp* '};' nl;

			# Ignore non-gl functions and fields.
			sp* ( 'static' sp+ )? ( 'const' sp+ )? [A-Za-z0-9_:]+ ( sp 'const'? | '*'+ ${ m4++ } )+ ( id - ( 'gl' id ) ) ( '(' cd* ')' )? sp* 'const'?
				sp* ( ';' | '{' cd* '}' ) sp* nl;

			# Ignore Q_DECLARE_FLAGS
			sp* 'Q_DECLARE_FLAGS(' cd+ ')' sp* nl;

			# Ignore deprecated functionality.
			'#if QT_DEPRECATED_SINCE(' cd+ ')' sp* nl
				( cd* - '#endif' ) sp* nl
				'#endif' sp* nl;

			# Done.
			sp* '}' sp* ';' nl
			{
				foundclass++;
				fgoto main;
			};

			# Reset relevant states on empty lines.
			sp* nl
			{
				// Reset heading comment.
				heading = ""

				// Start new line block.
				lineblock++
			};

		*|;

		skiperror := [^\n]* (';' | nl ) @{ fgoto main; };

		write init;
		write exec;
	}%%

	if p < pe {
		m0, m1 = p, p
		for m0 > 0 && data[m0-1] != '\n' {
			m0--
		}
		for m1 < len(data) && data[m1] != '\n' {
			m1++
		}
		return fmt.Errorf("cannot parse header file:%d:%d: %s", curline, p-m0, data[m0:m1])
	}

	if foundclass == 0 {
		return fmt.Errorf("cannot find C++ class in header file")
	}
	if foundclass == 1 {
		return fmt.Errorf("cannot find end of C++ class in header file")
	}
	if foundclass > 2 {
		return fmt.Errorf("found too many C++ classes in header file")
	}
	return nil
}
