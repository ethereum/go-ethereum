import { Token as AntlrToken } from 'antlr4';
import { Token, TokenizeOptions } from './types';
import type { Comment } from './ast-types';
export declare function buildTokenList(tokensArg: AntlrToken[], options: TokenizeOptions): Token[];
export declare function buildCommentList(tokensArg: AntlrToken[], commentsChannelId: number, options: TokenizeOptions): Comment[];
