#include <stdio.h>
#include <iostream>
#include <vector>
#include <map>
#include "util.h"
#include "parser.h"
#include "tokenize.h"

// Extended BEDMAS precedence order
int precedence(Node tok) {
    std::string v = tok.val;
    if (v == ".") return -1;
    else if (v == "!" || v == "not") return 1;
    else if (v=="^" || v == "**") return 2;
	else if (v=="*" || v=="/" || v=="%") return 3;
    else if (v=="+" || v=="-") return 4;
    else if (v=="<" || v==">" || v=="<=" || v==">=") return 5;
    else if (v=="&" || v=="|" || v=="xor" || v=="==" || v == "!=") return 6;
    else if (v=="&&" || v=="and") return 7;    
    else if (v=="||" || v=="or") return 8;
    else if (v=="=") return 10;
    else if (v=="+=" || v=="-=" || v=="*=" || v=="/=" || v=="%=") return 10;
    else if (v==":" || v == "::") return 11;
    else return 0;
}

// Token classification for shunting-yard purposes
int toktype(Node tok) {
    if (tok.type == ASTNODE) return COMPOUND;
    std::string v = tok.val;
    if (v == "(" || v == "[" || v == "{") return LPAREN;
    else if (v == ")" || v == "]" || v == "}") return RPAREN;
    else if (v == ",") return COMMA;
    else if (v == "!" || v == "~" || v == "not") return UNARY_OP;
    else if (precedence(tok) > 0) return BINARY_OP;
    else if (precedence(tok) < 0) return TOKEN_SPLITTER;
    if (tok.val[0] != '"' && tok.val[0] != '\'') {
		for (unsigned i = 0; i < tok.val.length(); i++) {
            if (chartype(tok.val[i]) == SYMB) {
                err("Invalid symbol: "+tok.val, tok.metadata);
            }
        }
    }
    return ALPHANUM;
}


// Converts to reverse polish notation
std::vector<Node> shuntingYard(std::vector<Node> tokens) {
    std::vector<Node> iq;
    for (int i = tokens.size() - 1; i >= 0; i--) {
        iq.push_back(tokens[i]);
    }
    std::vector<Node> oq;
    std::vector<Node> stack;
    Node prev, tok;
    int prevtyp = 0, toktyp = 0;
    
    while (iq.size()) {
        prev = tok;
        prevtyp = toktyp;
        tok = iq.back();
        toktyp = toktype(tok);
        iq.pop_back();
        // Alphanumerics go straight to output queue
        if (toktyp == ALPHANUM) {
            oq.push_back(tok);
        }
        // Left parens go on stack and output queue
        else if (toktyp == LPAREN) {
            while (stack.size() && toktype(stack.back()) == TOKEN_SPLITTER) {
                oq.push_back(stack.back());
                stack.pop_back();
            }
            if (prevtyp != ALPHANUM && prevtyp != RPAREN) {
                oq.push_back(token("id", tok.metadata));
            }
            stack.push_back(tok);
            oq.push_back(tok);
        }
        // If rparen, keep moving from stack to output queue until lparen
        else if (toktyp == RPAREN) {
            while (stack.size() && toktype(stack.back()) != LPAREN) {
                oq.push_back(stack.back());
                stack.pop_back();
            }
            if (stack.size()) {
                stack.pop_back();
            }
            oq.push_back(tok);
        }
        else if (toktyp == UNARY_OP) {
            stack.push_back(tok);
        }
        // If token splitter, just push it to the stack 
        else if (toktyp == TOKEN_SPLITTER) {
            while (stack.size() && toktype(stack.back()) == TOKEN_SPLITTER) {
                oq.push_back(stack.back());
                stack.pop_back();
            }
            stack.push_back(tok);
        }
        // If binary op, keep popping from stack while higher bedmas precedence
        else if (toktyp == BINARY_OP) {
            if (tok.val == "-" && prevtyp != ALPHANUM && prevtyp != RPAREN) {
                stack.push_back(tok);
                oq.push_back(token("0", tok.metadata));
            }
            else {
                int prec = precedence(tok);
                while (stack.size() 
                      && (toktype(stack.back()) == BINARY_OP 
                          || toktype(stack.back()) == UNARY_OP
                          || toktype(stack.back()) == TOKEN_SPLITTER)
                      && precedence(stack.back()) <= prec) {
                    oq.push_back(stack.back());
                    stack.pop_back();
                }
                stack.push_back(tok);
            }
        }
        // Comma means finish evaluating the argument
        else if (toktyp == COMMA) {
            while (stack.size() && toktype(stack.back()) != LPAREN) {
                oq.push_back(stack.back());
                stack.pop_back();
            }
        }
    }
    while (stack.size()) {
        oq.push_back(stack.back());
        stack.pop_back();
    }
    return oq;
}

// Converts reverse polish notation into tree
Node treefy(std::vector<Node> stream) {
    std::vector<Node> iq;
    for (int i = stream.size() -1; i >= 0; i--) {
        iq.push_back(stream[i]);
    }
    std::vector<Node> oq;
    while (iq.size()) {
        Node tok = iq.back();
        iq.pop_back();
        int typ = toktype(tok);
        // If unary, take node off end of oq and wrap it with the operator
        // If binary, do the same with two nodes
        if (typ == UNARY_OP || typ == BINARY_OP || typ == TOKEN_SPLITTER) {
            std::vector<Node> args;
            int rounds = (typ == UNARY_OP) ? 1 : 2;
            for (int i = 0; i < rounds; i++) {
                if (oq.size() == 0) {
                    err("Line malformed, not enough args for "+tok.val,
                        tok.metadata);
                }
                args.push_back(oq.back());
                oq.pop_back();
            }
            std::vector<Node> args2;
            while (args.size()) {
                args2.push_back(args.back());
                args.pop_back();
            }
            oq.push_back(astnode(tok.val, args2, tok.metadata));
        }
        // If rparen, keep grabbing until we get to an lparen
        else if (typ == RPAREN) {
            std::vector<Node> args;
            while (1) {
                if (toktype(oq.back()) == LPAREN) break;
                args.push_back(oq.back());
                oq.pop_back();
                if (!oq.size()) err("Bracket without matching", tok.metadata);
            }
            oq.pop_back();
            args.push_back(oq.back());
            oq.pop_back();
            // We represent a[b] as (access a b)
            if (tok.val == "]")
                 args.push_back(token("access", tok.metadata));
            if (args.back().type == ASTNODE)
                 args.push_back(token("fun", tok.metadata));
            std::string fun = args.back().val;
            args.pop_back();
            // We represent [1,2,3] as (array_lit 1 2 3)
            if (fun == "access" && args.size() && args.back().val == "id") {
                fun = "array_lit";
                args.pop_back();
            }
            std::vector<Node> args2;
            while (args.size()) {
                args2.push_back(args.back());
                args.pop_back();
            }
            // When evaluating 2 + (3 * 5), the shunting yard algo turns that
            // into 2 ( id 3 5 * ) +, effectively putting "id" as a dummy
            // function where the algo was expecting a function to call the
            // thing inside the brackets. This reverses that step
			if (fun == "id" && args2.size() == 1) {
                oq.push_back(args2[0]);
            }
            else {
                oq.push_back(astnode(fun, args2, tok.metadata));
            }
        }
        else oq.push_back(tok);
        // This is messy, but has to be done. Import/inset other files here
        std::string v = oq.back().val;
        if ((v == "inset" || v == "import" || v == "create") 
                && oq.back().args.size() == 1
                && oq.back().args[0].type == TOKEN) {
            int lastSlashPos = tok.metadata.file.rfind("/");
            std::string root;
            if (lastSlashPos >= 0)
                root = tok.metadata.file.substr(0, lastSlashPos) + "/";
            else
                root = "";
            std::string filename = oq.back().args[0].val;
            filename = filename.substr(1, filename.length() - 2);
            if (!exists(root + filename))
                err("File does not exist: "+root + filename, tok.metadata);
            oq.back().args.pop_back();
            oq.back().args.push_back(parseSerpent(root + filename));
        }
        //Useful for debugging
        //for (int i = 0; i < oq.size(); i++) {
        //    std::cerr << printSimple(oq[i]) << " ";
        //}
        //std::cerr << " <-\n";
    }
    // Output must have one argument
    if (oq.size() == 0) {
        err("Output blank", Metadata());
    }
    else if (oq.size() > 1) {
        return asn("multi", oq, oq[0].metadata);
    }

	return oq[0];
}


// Parses one line of serpent
Node parseSerpentTokenStream(std::vector<Node> s) {
    return treefy(shuntingYard(s));
}


// Count spaces at beginning of line
int spaceCount(std::string s) {
	unsigned pos = 0;
	while (pos < s.length() && (s[pos] == ' ' || s[pos] == '\t'))
		pos++;
    return pos;
}

// Is this a command that takes an argument on the same line?
bool bodied(std::string tok) {
    return tok == "if" || tok == "elif" || tok == "while"
        || tok == "with" || tok == "def" || tok == "extern"
        || tok == "data" || tok == "assert" || tok == "return"
        || tok == "fun" || tok == "scope" || tok == "macro"
        || tok == "type";
}

// Are the two commands meant to continue each other? 
bool bodiedContinued(std::string prev, std::string tok) {
    return (prev == "if" && tok == "elif")
        || (prev == "elif" && tok == "else")
        || (prev == "elif" && tok == "elif")
        || (prev == "if" && tok == "else");
}

// Is a line of code empty?
bool isLineEmpty(std::string line) {
    std::vector<Node> tokens = tokenize(line);
    if (!tokens.size() || tokens[0].val == "#" || tokens[0].val == "//")
        return true;
    return false;
}

// Parse lines of serpent (helper function)
Node parseLines(std::vector<std::string> lines, Metadata metadata, int sp) {
    std::vector<Node> o;
    int origLine = metadata.ln;
	unsigned i = 0;
    while (i < lines.size()) {
        metadata.ln = origLine + i; 
        std::string main = lines[i];
        if (isLineEmpty(main)) {
            i += 1;
            continue;
        }
        int spaces = spaceCount(main);
        if (spaces != sp) {
            err("Indent mismatch", metadata);
        }
        // Tokenize current line
        std::vector<Node> tokens = tokenize(main.substr(sp), metadata);
        // Remove comments
        std::vector<Node> tokens2;
		for (unsigned j = 0; j < tokens.size(); j++) {
            if (tokens[j].val == "#" || tokens[j].val == "//") break;
            tokens2.push_back(tokens[j]);
        }
        bool expectingChildBlock = false;
        if (tokens2.size() > 0 && tokens2.back().val == ":") {
            tokens2.pop_back();
            expectingChildBlock = true;
        }
        // Parse current line
        Node out = parseSerpentTokenStream(tokens2);
        // Parse child block
        int childIndent = 999999;
        std::vector<std::string> childBlock;
        while (1) {
			i++;
			if (i >= lines.size())
				break;
            bool ile = isLineEmpty(lines[i]);
            if (!ile) {
                int spaces = spaceCount(lines[i]);
                if (spaces <= sp) break;
                childBlock.push_back(lines[i]);
                if (spaces < childIndent) childIndent = spaces;
            }
            else childBlock.push_back("");
        }
        // Child block empty?
        bool cbe = true;
		for (unsigned i = 0; i < childBlock.size(); i++) {
            if (childBlock[i].length() > 0) { cbe = false; break; }
        }
        // Add child block to AST
        if (expectingChildBlock) {
            if (cbe)
                err("Expected indented child block!", out.metadata);
            out.type = ASTNODE;
            metadata.ln += 1;
            out.args.push_back(parseLines(childBlock, metadata, childIndent));
            metadata.ln -= 1;
        }
        else if (!cbe)
            err("Did not expect indented child block!", out.metadata);
        else if (out.args.size() && out.args[out.args.size() - 1].val == ":") {
            Node n = out.args[out.args.size() - 1];
            out.args.pop_back();
            out.args.push_back(n.args[0]);
            out.args.push_back(n.args[1]);
        }
        // Bring back if / elif into AST
        if (bodied(tokens[0].val)) {
            if (out.val != "multi") {
                // token not being used in bodied form
            }
            else if (out.args[0].val == "id")
                out = astnode(tokens[0].val, out.args[1].args, out.metadata);
            else if (out.args[0].type == TOKEN) {
                std::vector<Node> out2;
                for (unsigned i = 1; i < out.args.size(); i++)
                    out2.push_back(out.args[i]);
                out = astnode(tokens[0].val, out2, out.metadata);
            }
            else
                out = astnode("fun", out.args, out.metadata);
        }
        // Multi not supported
        if (out.val == "multi")
            err("Multiple expressions or unclosed bracket", out.metadata);
        // Convert top-level colon expressions into non-colon expressions;
        // makes if statements and the like equivalent indented or not
        //if (out.val == ":" && out.args[0].type == TOKEN)
        //    out = asn(out.args[0].val, out.args[1], out.metadata);
        //if (bodied(tokens[0].val) && out.args[0].val == ":")
        //    out = asn(tokens[0].val, out.args[0].args);
        if (o.size() == 0 || o.back().type == TOKEN) {
            o.push_back(out);
            continue;
        }
        // This is a little complicated. Basically, the idea here is to build
        // constructions like [if [< x 5] [a] [elif [< x 10] [b] [else [c]]]]
        std::vector<Node> u;
        u.push_back(o.back());
        if (bodiedContinued(o.back().val, out.val)) {
            while (1) {
                if (!bodiedContinued(u.back().val, out.val)) {
                    u.pop_back();
                    break;
                }
                if (!u.back().args.size()
                 || !bodiedContinued(u.back().val, u.back().args.back().val)) {
                    break;
                }
                u.push_back(u.back().args.back());
            }
            u.back().args.push_back(out);
            while (u.size() > 1) {
                Node v = u.back();
                u.pop_back();
                u.back().args.pop_back();
                u.back().args.push_back(v);
            }
            o.pop_back();
            o.push_back(u[0]);
        }
        else o.push_back(out);
    }
	if (o.size() == 1)
		return o[0];
	else if (o.size())
		return astnode("seq", o, o[0].metadata);
	else
		return astnode("seq", o, Metadata());
}

// Parses serpent code
Node parseSerpent(std::string s) {
    std::string input = s;
    std::string file = "main";
    if (exists(s)) {
        file = s;
        input = get_file_contents(s);
    }
    return parseLines(splitLines(input), Metadata(file, 0, 0), 0);
}


using namespace std;
