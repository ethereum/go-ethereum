#![deny(clippy::all)]

#[macro_use]
extern crate napi_derive;

use napi::Error;
use solang_parser::lexer::{Lexer, Token};
use std::result::Result;

enum State {
  TopLevel,
  IgnoringStatement,
  CurlyBracesOpened(usize),
  PragmaFound,
  PragmaSolidityFound,
  ImportFound,
  ImportStarFound,
  ImportStarAsFound,
  ImportStarAsIdentifierFound,
  ImportStarAsIdentifierFromFound,
  ImportCurlyBracesOpened(usize),
  ImportSimbolAliasesFound,
  ImportSimbolAliasesFromFound,
}

#[napi(object)]
pub struct AnalysisResult {
  pub version_pragmas: Vec<String>,
  pub imports: Vec<String>,
}

#[napi]
pub fn analyze(input: String) -> Result<AnalysisResult, Error> {
  let mut comments = Vec::new();
  let lexer = Lexer::new(&input, 0, &mut comments);

  let mut version_pragmas = Vec::new();
  let mut imports = Vec::new();

  let mut state = State::TopLevel;

  for item in lexer {
    if item.is_err() {
      continue;
    }

    let (_, token, _) = item.unwrap();

    match state {
      State::TopLevel => match token {
        Token::Pragma => {
          state = State::PragmaFound;
        }
        Token::Import => {
          state = State::ImportFound;
        }
        Token::OpenCurlyBrace => {
          state = State::CurlyBracesOpened(1);
        }
        Token::Semicolon => {
          state = State::TopLevel;
        }
        Token::DocComment(_, _) => {
          // Do nothing
        }
        _ => {
          state = State::IgnoringStatement;
        }
      },
      State::IgnoringStatement => match token {
        Token::OpenCurlyBrace => {
          state = State::CurlyBracesOpened(1);
        }
        Token::Semicolon => {
          state = State::TopLevel;
        }
        _ => {}
      },
      State::CurlyBracesOpened(braces) => match token {
        Token::OpenCurlyBrace => {
          state = State::CurlyBracesOpened(braces + 1);
        }
        Token::CloseCurlyBrace => {
          if braces == 1 {
            state = State::TopLevel;
          } else {
            state = State::CurlyBracesOpened(braces - 1);
          }
        }
        _ => {}
      },
      State::PragmaFound => match token {
        Token::Identifier(id) => {
          if id == "solidity" {
            state = State::PragmaSolidityFound;
          } else {
            state = State::IgnoringStatement;
          }
        }
        Token::OpenCurlyBrace => {
          state = State::CurlyBracesOpened(1);
        }
        Token::Semicolon => {
          state = State::TopLevel;
        }
        _ => {
          state = State::IgnoringStatement;
        }
      },
      State::PragmaSolidityFound => match token {
        Token::StringLiteral(literal) => {
          version_pragmas.push(literal.replace(['\r', '\n'], ""));
          state = State::IgnoringStatement;
        }
        Token::OpenCurlyBrace => {
          state = State::CurlyBracesOpened(1);
        }
        Token::Semicolon => {
          state = State::TopLevel;
        }
        _ => {
          state = State::IgnoringStatement;
        }
      },
      State::ImportFound => match token {
        Token::StringLiteral(literal) => {
          imports.push(literal.to_string());
          state = State::IgnoringStatement;
        }
        Token::Mul => {
          state = State::ImportStarFound;
        }
        Token::OpenCurlyBrace => {
          state = State::ImportCurlyBracesOpened(1);
        }
        Token::Semicolon => {
          state = State::TopLevel;
        }
        _ => {
          state = State::IgnoringStatement;
        }
      },
      State::ImportStarFound => match token {
        Token::As => {
          state = State::ImportStarAsFound;
        }
        Token::OpenCurlyBrace => {
          state = State::CurlyBracesOpened(1);
        }
        Token::Semicolon => {
          state = State::TopLevel;
        }
        _ => {
          state = State::IgnoringStatement;
        }
      },
      State::ImportStarAsFound => match token {
        Token::Identifier(_) => {
          state = State::ImportStarAsIdentifierFound;
        }
        Token::OpenCurlyBrace => {
          state = State::CurlyBracesOpened(1);
        }
        Token::Semicolon => {
          state = State::TopLevel;
        }
        _ => {
          state = State::IgnoringStatement;
        }
      },
      State::ImportStarAsIdentifierFound => match token {
        Token::Identifier("from") => {
          state = State::ImportStarAsIdentifierFromFound;
        }
        Token::OpenCurlyBrace => {
          state = State::CurlyBracesOpened(1);
        }
        Token::Semicolon => {
          state = State::TopLevel;
        }
        _ => {
          state = State::IgnoringStatement;
        }
      },
      State::ImportStarAsIdentifierFromFound => match token {
        Token::StringLiteral(literal) => {
          imports.push(literal.to_string());
          state = State::IgnoringStatement;
        }
        Token::OpenCurlyBrace => {
          state = State::CurlyBracesOpened(1);
        }
        Token::Semicolon => {
          state = State::TopLevel;
        }
        _ => {
          state = State::IgnoringStatement;
        }
      },
      State::ImportCurlyBracesOpened(braces) => match token {
        Token::OpenCurlyBrace => {
          state = State::CurlyBracesOpened(braces + 1);
        }
        Token::CloseCurlyBrace => {
          if braces == 1 {
            state = State::ImportSimbolAliasesFound;
          } else {
            state = State::CurlyBracesOpened(braces - 1);
          }
        }
        Token::Semicolon => {
          state = State::TopLevel;
        }
        _ => {}
      },
      State::ImportSimbolAliasesFound => match token {
        Token::Identifier("from") => {
          state = State::ImportSimbolAliasesFromFound;
        }
        Token::OpenCurlyBrace => {
          state = State::CurlyBracesOpened(1);
        }
        Token::Semicolon => {
          state = State::TopLevel;
        }
        _ => {
          state = State::IgnoringStatement;
        }
      },
      State::ImportSimbolAliasesFromFound => match token {
        Token::StringLiteral(literal) => {
          imports.push(literal.to_string());
          state = State::IgnoringStatement;
        }
        Token::OpenCurlyBrace => {
          state = State::CurlyBracesOpened(1);
        }
        Token::Semicolon => {
          state = State::TopLevel;
        }
        _ => {
          state = State::IgnoringStatement;
        }
      },
    }
  }

  let res = AnalysisResult {
    version_pragmas,
    imports,
  };

  Ok(res)
}
