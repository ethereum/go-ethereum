"use strict";
// Generated from XPathLexer.g4 by ANTLR 4.9.0-SNAPSHOT
Object.defineProperty(exports, "__esModule", { value: true });
exports.XPathLexer = void 0;
const ATNDeserializer_1 = require("../../atn/ATNDeserializer");
const Lexer_1 = require("../../Lexer");
const LexerATNSimulator_1 = require("../../atn/LexerATNSimulator");
const VocabularyImpl_1 = require("../../VocabularyImpl");
const Utils = require("../../misc/Utils");
class XPathLexer extends Lexer_1.Lexer {
    // tslint:enable:no-trailing-whitespace
    constructor(input) {
        super(input);
        this._interp = new LexerATNSimulator_1.LexerATNSimulator(XPathLexer._ATN, this);
    }
    // @Override
    // @NotNull
    get vocabulary() {
        return XPathLexer.VOCABULARY;
    }
    // @Override
    get grammarFileName() { return "XPathLexer.g4"; }
    // @Override
    get ruleNames() { return XPathLexer.ruleNames; }
    // @Override
    get serializedATN() { return XPathLexer._serializedATN; }
    // @Override
    get channelNames() { return XPathLexer.channelNames; }
    // @Override
    get modeNames() { return XPathLexer.modeNames; }
    // @Override
    action(_localctx, ruleIndex, actionIndex) {
        switch (ruleIndex) {
            case 4:
                this.ID_action(_localctx, actionIndex);
                break;
        }
    }
    ID_action(_localctx, actionIndex) {
        switch (actionIndex) {
            case 0:
                let text = this.text;
                if (text.charAt(0) === text.charAt(0).toUpperCase()) {
                    this.type = XPathLexer.TOKEN_REF;
                }
                else {
                    this.type = XPathLexer.RULE_REF;
                }
                break;
        }
    }
    static get _ATN() {
        if (!XPathLexer.__ATN) {
            XPathLexer.__ATN = new ATNDeserializer_1.ATNDeserializer().deserialize(Utils.toCharArray(XPathLexer._serializedATN));
        }
        return XPathLexer.__ATN;
    }
}
exports.XPathLexer = XPathLexer;
XPathLexer.TOKEN_REF = 1;
XPathLexer.RULE_REF = 2;
XPathLexer.ANYWHERE = 3;
XPathLexer.ROOT = 4;
XPathLexer.WILDCARD = 5;
XPathLexer.BANG = 6;
XPathLexer.ID = 7;
XPathLexer.STRING = 8;
// tslint:disable:no-trailing-whitespace
XPathLexer.channelNames = [
    "DEFAULT_TOKEN_CHANNEL", "HIDDEN",
];
// tslint:disable:no-trailing-whitespace
XPathLexer.modeNames = [
    "DEFAULT_MODE",
];
XPathLexer.ruleNames = [
    "ANYWHERE", "ROOT", "WILDCARD", "BANG", "ID", "NameChar", "NameStartChar",
    "STRING",
];
XPathLexer._LITERAL_NAMES = [
    undefined, undefined, undefined, "'//'", "'/'", "'*'", "'!'",
];
XPathLexer._SYMBOLIC_NAMES = [
    undefined, "TOKEN_REF", "RULE_REF", "ANYWHERE", "ROOT", "WILDCARD", "BANG",
    "ID", "STRING",
];
XPathLexer.VOCABULARY = new VocabularyImpl_1.VocabularyImpl(XPathLexer._LITERAL_NAMES, XPathLexer._SYMBOLIC_NAMES, []);
XPathLexer._serializedATNSegments = 2;
XPathLexer._serializedATNSegment0 = "\x03\uC91D\uCABA\u058D\uAFBA\u4F53\u0607\uEA8B\uC241\x02\n2\b\x01\x04" +
    "\x02\t\x02\x04\x03\t\x03\x04\x04\t\x04\x04\x05\t\x05\x04\x06\t\x06\x04" +
    "\x07\t\x07\x04\b\t\b\x04\t\t\t\x03\x02\x03\x02\x03\x02\x03\x03\x03\x03" +
    "\x03\x04\x03\x04\x03\x05\x03\x05\x03\x06\x03\x06\x07\x06\x1F\n\x06\f\x06" +
    "\x0E\x06\"\v\x06\x03\x06\x03\x06\x03\x07\x03\x07\x03\b\x03\b\x03\t\x03" +
    "\t\x07\t,\n\t\f\t\x0E\t/\v\t\x03\t\x03\t\x03-\x02\x02\n\x03\x02\x05\x05" +
    "\x02\x06\x07\x02\x07\t\x02\b\v\x02\t\r\x02\x02\x0F\x02\x02\x11\x02\n\x03" +
    "\x02\x02\x04\u02B6\x02\x02\x02\n\x02\x10\x02\x1D\x022\x02;\x02C\x02\\" +
    "\x02a\x02a\x02c\x02|\x02\x81\x02\xA1\x02\xAC\x02\xAC\x02\xAF\x02\xAF\x02" +
    "\xB7\x02\xB7\x02\xBC\x02\xBC\x02\xC2\x02\xD8\x02\xDA\x02\xF8\x02\xFA\x02" +
    "\u02C3\x02\u02C8\x02\u02D3\x02\u02E2\x02\u02E6\x02\u02EE\x02\u02EE\x02" +
    "\u02F0\x02\u02F0\x02\u0302\x02\u0376\x02\u0378\x02\u0379\x02\u037C\x02" +
    "\u037F\x02\u0381\x02\u0381\x02\u0388\x02\u0388\x02\u038A\x02\u038C\x02" +
    "\u038E\x02\u038E\x02\u0390\x02\u03A3\x02\u03A5\x02\u03F7\x02\u03F9\x02" +
    "\u0483\x02\u0485\x02\u0489\x02\u048C\x02\u0531\x02\u0533\x02\u0558\x02" +
    "\u055B\x02\u055B\x02\u0563\x02\u0589\x02\u0593\x02\u05BF\x02\u05C1\x02" +
    "\u05C1\x02\u05C3\x02\u05C4\x02\u05C6\x02\u05C7\x02\u05C9\x02\u05C9\x02" +
    "\u05D2\x02\u05EC\x02\u05F2\x02\u05F4\x02\u0602\x02\u0607\x02\u0612\x02" +
    "\u061C\x02\u061E\x02\u061E\x02\u0622\x02\u066B\x02\u0670\x02\u06D5\x02" +
    "\u06D7\x02\u06DF\x02\u06E1\x02\u06EA\x02\u06EC\x02\u06FE\x02\u0701\x02" +
    "\u0701\x02\u0711\x02\u074C\x02\u074F\x02\u07B3\x02\u07C2\x02\u07F7\x02" +
    "\u07FC\x02\u07FC\x02\u0802\x02\u082F\x02\u0842\x02\u085D\x02\u08A2\x02" +
    "\u08B6\x02\u08B8\x02\u08BF\x02\u08D6\x02\u0965\x02\u0968\x02\u0971\x02" +
    "\u0973\x02\u0985\x02\u0987\x02\u098E\x02\u0991\x02\u0992\x02\u0995\x02" +
    "\u09AA\x02\u09AC\x02\u09B2\x02\u09B4\x02\u09B4\x02\u09B8\x02\u09BB\x02" +
    "\u09BE\x02\u09C6\x02\u09C9\x02\u09CA\x02\u09CD\x02\u09D0\x02\u09D9\x02" +
    "\u09D9\x02\u09DE\x02\u09DF\x02\u09E1\x02\u09E5\x02\u09E8\x02\u09F3\x02" +
    "\u0A03\x02\u0A05\x02\u0A07\x02\u0A0C\x02\u0A11\x02\u0A12\x02\u0A15\x02" +
    "\u0A2A\x02\u0A2C\x02\u0A32\x02\u0A34\x02\u0A35\x02\u0A37\x02\u0A38\x02" +
    "\u0A3A\x02\u0A3B\x02\u0A3E\x02\u0A3E\x02\u0A40\x02\u0A44\x02\u0A49\x02" +
    "\u0A4A\x02\u0A4D\x02\u0A4F\x02\u0A53\x02\u0A53\x02\u0A5B\x02\u0A5E\x02" +
    "\u0A60\x02\u0A60\x02\u0A68\x02\u0A77\x02\u0A83\x02\u0A85\x02\u0A87\x02" +
    "\u0A8F\x02\u0A91\x02\u0A93\x02\u0A95\x02\u0AAA\x02\u0AAC\x02\u0AB2\x02" +
    "\u0AB4\x02\u0AB5\x02\u0AB7\x02\u0ABB\x02\u0ABE\x02\u0AC7\x02\u0AC9\x02" +
    "\u0ACB\x02\u0ACD\x02\u0ACF\x02\u0AD2\x02\u0AD2\x02\u0AE2\x02\u0AE5\x02" +
    "\u0AE8\x02\u0AF1\x02\u0AFB\x02\u0AFB\x02\u0B03\x02\u0B05\x02\u0B07\x02" +
    "\u0B0E\x02\u0B11\x02\u0B12\x02\u0B15\x02\u0B2A\x02\u0B2C\x02\u0B32\x02" +
    "\u0B34\x02\u0B35\x02\u0B37\x02\u0B3B\x02\u0B3E\x02\u0B46\x02\u0B49\x02" +
    "\u0B4A\x02\u0B4D\x02\u0B4F\x02\u0B58\x02\u0B59\x02\u0B5E\x02\u0B5F\x02" +
    "\u0B61\x02\u0B65\x02\u0B68\x02\u0B71\x02\u0B73\x02\u0B73\x02\u0B84\x02" +
    "\u0B85\x02\u0B87\x02\u0B8C\x02\u0B90\x02\u0B92\x02\u0B94\x02\u0B97\x02" +
    "\u0B9B\x02\u0B9C\x02\u0B9E\x02\u0B9E\x02\u0BA0\x02\u0BA1\x02\u0BA5\x02" +
    "\u0BA6\x02\u0BAA\x02\u0BAC\x02\u0BB0\x02\u0BBB\x02\u0BC0\x02\u0BC4\x02" +
    "\u0BC8\x02\u0BCA\x02\u0BCC\x02\u0BCF\x02\u0BD2\x02\u0BD2\x02\u0BD9\x02" +
    "\u0BD9\x02\u0BE8\x02\u0BF1\x02\u0C02\x02\u0C05\x02\u0C07\x02\u0C0E\x02" +
    "\u0C10\x02\u0C12\x02\u0C14\x02\u0C2A\x02\u0C2C\x02\u0C3B\x02\u0C3F\x02" +
    "\u0C46\x02\u0C48\x02\u0C4A\x02\u0C4C\x02\u0C4F\x02\u0C57\x02\u0C58\x02" +
    "\u0C5A\x02\u0C5C\x02\u0C62\x02\u0C65\x02\u0C68\x02\u0C71\x02\u0C82\x02" +
    "\u0C85\x02\u0C87\x02\u0C8E\x02\u0C90\x02\u0C92\x02\u0C94\x02\u0CAA\x02" +
    "\u0CAC\x02\u0CB5\x02\u0CB7\x02\u0CBB\x02\u0CBE\x02\u0CC6\x02\u0CC8\x02" +
    "\u0CCA\x02\u0CCC\x02\u0CCF\x02\u0CD7\x02\u0CD8\x02\u0CE0\x02\u0CE0\x02" +
    "\u0CE2\x02\u0CE5\x02\u0CE8\x02\u0CF1\x02\u0CF3\x02\u0CF4\x02\u0D03\x02" +
    "\u0D05\x02\u0D07\x02\u0D0E\x02\u0D10\x02\u0D12\x02\u0D14\x02\u0D3C\x02" +
    "\u0D3F\x02\u0D46\x02\u0D48\x02\u0D4A\x02\u0D4C\x02\u0D50\x02\u0D56\x02" +
    "\u0D59\x02\u0D61\x02\u0D65\x02\u0D68\x02\u0D71\x02\u0D7C\x02\u0D81\x02" +
    "\u0D84\x02\u0D85\x02\u0D87\x02\u0D98\x02\u0D9C\x02\u0DB3\x02\u0DB5\x02" +
    "\u0DBD\x02\u0DBF\x02\u0DBF\x02\u0DC2\x02\u0DC8\x02\u0DCC\x02\u0DCC\x02" +
    "\u0DD1\x02\u0DD6\x02\u0DD8\x02\u0DD8\x02\u0DDA\x02\u0DE1\x02\u0DE8\x02" +
    "\u0DF1\x02\u0DF4\x02\u0DF5\x02\u0E03\x02\u0E3C\x02\u0E42\x02\u0E50\x02" +
    "\u0E52\x02\u0E5B\x02\u0E83\x02\u0E84\x02\u0E86\x02\u0E86\x02\u0E89\x02" +
    "\u0E8A\x02\u0E8C\x02\u0E8C\x02\u0E8F\x02\u0E8F\x02\u0E96\x02\u0E99\x02" +
    "\u0E9B\x02\u0EA1\x02\u0EA3\x02\u0EA5\x02\u0EA7\x02\u0EA7\x02\u0EA9\x02" +
    "\u0EA9\x02\u0EAC\x02\u0EAD\x02\u0EAF\x02\u0EBB\x02\u0EBD\x02\u0EBF\x02" +
    "\u0EC2\x02\u0EC6\x02\u0EC8\x02\u0EC8\x02\u0ECA\x02\u0ECF\x02\u0ED2\x02" +
    "\u0EDB\x02\u0EDE\x02\u0EE1\x02\u0F02\x02\u0F02\x02\u0F1A\x02\u0F1B\x02" +
    "\u0F22\x02\u0F2B\x02\u0F37\x02\u0F37\x02\u0F39\x02\u0F39\x02\u0F3B\x02" +
    "\u0F3B\x02\u0F40\x02\u0F49\x02\u0F4B\x02\u0F6E\x02\u0F73\x02\u0F86\x02" +
    "\u0F88\x02\u0F99\x02\u0F9B\x02\u0FBE\x02\u0FC8\x02\u0FC8\x02\u1002\x02" +
    "\u104B\x02\u1052\x02\u109F\x02\u10A2\x02\u10C7\x02\u10C9\x02\u10C9\x02" +
    "\u10CF\x02\u10CF\x02\u10D2\x02\u10FC\x02\u10FE\x02\u124A\x02\u124C\x02" +
    "\u124F\x02\u1252\x02\u1258\x02\u125A\x02\u125A\x02\u125C\x02\u125F\x02" +
    "\u1262\x02\u128A\x02\u128C\x02\u128F\x02\u1292\x02\u12B2\x02\u12B4\x02" +
    "\u12B7\x02\u12BA\x02\u12C0\x02\u12C2\x02\u12C2\x02\u12C4\x02\u12C7\x02" +
    "\u12CA\x02\u12D8\x02\u12DA\x02\u1312\x02\u1314\x02\u1317\x02\u131A\x02" +
    "\u135C\x02\u135F\x02\u1361\x02\u1382\x02\u1391\x02\u13A2\x02\u13F7\x02" +
    "\u13FA\x02\u13FF\x02\u1403\x02\u166E\x02\u1671\x02\u1681\x02\u1683\x02" +
    "\u169C\x02\u16A2\x02\u16EC\x02\u16F0\x02\u16FA\x02\u1702\x02\u170E\x02" +
    "\u1710\x02\u1716\x02\u1722\x02\u1736\x02\u1742\x02\u1755\x02\u1762\x02" +
    "\u176E\x02\u1770\x02\u1772\x02\u1774\x02\u1775\x02\u1782\x02\u17D5\x02" +
    "\u17D9\x02\u17D9\x02\u17DE\x02\u17DF\x02\u17E2\x02\u17EB\x02\u180D\x02" +
    "\u1810\x02\u1812\x02\u181B\x02\u1822\x02\u1879\x02\u1882\x02\u18AC\x02" +
    "\u18B2\x02\u18F7\x02\u1902\x02\u1920\x02\u1922\x02\u192D\x02\u1932\x02" +
    "\u193D\x02\u1948\x02\u196F\x02\u1972\x02\u1976\x02\u1982\x02\u19AD\x02" +
    "\u19B2\x02\u19CB\x02\u19D2\x02\u19DB\x02\u1A02\x02\u1A1D\x02\u1A22\x02" +
    "\u1A60\x02\u1A62\x02\u1A7E\x02\u1A81\x02\u1A8B\x02\u1A92\x02\u1A9B\x02" +
    "\u1AA9\x02\u1AA9\x02\u1AB2\x02\u1ABF\x02\u1B02\x02\u1B4D\x02\u1B52\x02" +
    "\u1B5B\x02\u1B6D\x02\u1B75\x02\u1B82\x02\u1BF5\x02\u1C02\x02\u1C39\x02" +
    "\u1C42\x02\u1C4B\x02\u1C4F\x02\u1C7F\x02\u1C82\x02\u1C8A\x02\u1CD2\x02" +
    "\u1CD4\x02\u1CD6\x02\u1CF8\x02\u1CFA\x02\u1CFB\x02\u1D02\x02\u1DF7\x02" +
    "\u1DFD\x02\u1F17\x02\u1F1A\x02\u1F1F\x02\u1F22\x02\u1F47\x02\u1F4A\x02" +
    "\u1F4F\x02\u1F52\x02\u1F59\x02\u1F5B\x02\u1F5B\x02\u1F5D\x02\u1F5D\x02" +
    "\u1F5F\x02\u1F5F\x02\u1F61\x02\u1F7F\x02\u1F82\x02\u1FB6\x02\u1FB8\x02" +
    "\u1FBE\x02\u1FC0\x02\u1FC0\x02\u1FC4\x02\u1FC6\x02\u1FC8\x02\u1FCE\x02" +
    "\u1FD2\x02\u1FD5\x02\u1FD8\x02\u1FDD\x02\u1FE2\x02\u1FEE\x02\u1FF4\x02" +
    "\u1FF6\x02\u1FF8\x02\u1FFE\x02\u200D\x02\u2011\x02\u202C\x02\u2030\x02" +
    "\u2041\x02\u2042\x02\u2056\x02\u2056\x02\u2062\x02\u2066\x02\u2068\x02" +
    "\u2071\x02\u2073\x02\u2073\x02\u2081\x02\u2081\x02\u2092\x02\u209E\x02" +
    "\u20D2\x02\u20DE\x02\u20E3\x02\u20E3\x02\u20E7\x02\u20F2\x02\u2104\x02" +
    "\u2104\x02\u2109\x02\u2109\x02\u210C\x02\u2115\x02\u2117\x02\u2117\x02" +
    "\u211B\x02\u211F\x02\u2126\x02\u2126\x02\u2128\x02\u2128\x02\u212A\x02" +
    "\u212A\x02\u212C\x02\u212F\x02\u2131\x02\u213B\x02\u213E\x02\u2141\x02" +
    "\u2147\x02\u214B\x02\u2150\x02\u2150\x02\u2162\x02\u218A\x02\u2C02\x02" +
    "\u2C30\x02\u2C32\x02\u2C60\x02\u2C62\x02\u2CE6\x02\u2CED\x02\u2CF5\x02" +
    "\u2D02\x02\u2D27\x02\u2D29\x02\u2D29\x02\u2D2F\x02\u2D2F\x02\u2D32\x02" +
    "\u2D69\x02\u2D71\x02\u2D71\x02\u2D81\x02\u2D98\x02\u2DA2\x02\u2DA8\x02" +
    "\u2DAA\x02\u2DB0\x02\u2DB2\x02\u2DB8\x02\u2DBA\x02\u2DC0\x02\u2DC2\x02" +
    "\u2DC8\x02\u2DCA\x02\u2DD0\x02\u2DD2\x02\u2DD8\x02\u2DDA\x02\u2DE0\x02" +
    "\u2DE2\x02\u2E01\x02\u2E31\x02\u2E31\x02\u3007\x02\u3009\x02\u3023\x02" +
    "\u3031\x02\u3033\x02\u3037\x02\u303A\x02\u303E\x02\u3043\x02\u3098\x02" +
    "\u309B\x02\u309C\x02\u309F\x02\u30A1\x02\u30A3\x02\u30FC\x02\u30FE\x02" +
    "\u3101\x02\u3107\x02\u312F\x02\u3133\x02\u3190\x02\u31A2\x02\u31BC\x02" +
    "\u31F2\x02\u3201\x02\u3402\x02\u4DB7\x02\u4E02\x02\u9FD7\x02\uA002\x02" +
    "\uA48E\x02\uA4D2\x02\uA4FF\x02\uA502\x02\uA60E\x02\uA612\x02\uA62D\x02" +
    "\uA642\x02\uA671\x02\uA676\x02\uA67F\x02\uA681\x02\uA6F3\x02\uA719\x02" +
    "\uA721\x02\uA724\x02\uA78A\x02\uA78D\x02\uA7B0\x02\uA7B2\x02\uA7B9\x02" +
    "\uA7F9\x02\uA829\x02\uA842\x02\uA875\x02\uA882\x02\uA8C7\x02\uA8D2\x02" +
    "\uA8DB\x02\uA8E2\x02\uA8F9\x02\uA8FD\x02\uA8FD\x02\uA8FF\x02\uA8FF\x02" +
    "\uA902\x02\uA92F\x02\uA932\x02\uA955\x02\uA962\x02\uA97E\x02\uA982\x02" +
    "\uA9C2\x02\uA9D1\x02\uA9DB\x02\uA9E2\x02\uAA00\x02\uAA02\x02\uAA38\x02" +
    "\uAA42\x02\uAA4F\x02\uAA52\x02\uAA5B\x02\uAA62\x02\uAA78\x02\uAA7C\x02" +
    "\uAAC4\x02\uAADD\x02\uAADF\x02\uAAE2\x02\uAAF1\x02\uAAF4\x02\uAAF8\x02" +
    "\uAB03\x02\uAB08\x02\uAB0B\x02\uAB10\x02\uAB13\x02\uAB18\x02\uAB22\x02" +
    "\uAB28\x02\uAB2A\x02\uAB30\x02\uAB32\x02\uAB5C\x02\uAB5E\x02\uAB67\x02" +
    "\uAB72\x02\uABEC\x02\uABEE\x02\uABEF\x02\uABF2\x02\uABFB\x02\uAC02\x02" +
    "\uD7A5\x02\uD7B2\x02\uD7C8\x02\uD7CD\x02\uD7FD\x02\uF902\x02\uFA6F\x02" +
    "\uFA72\x02\uFADB\x02\uFB02\x02\uFB08\x02\uFB15\x02\uFB19\x02\uFB1F\x02" +
    "\uFB2A\x02\uFB2C\x02\uFB38\x02\uFB3A\x02\uFB3E\x02\uFB40\x02\uFB40\x02" +
    "\uFB42\x02\uFB43\x02\uFB45\x02\uFB46\x02\uFB48\x02\uFBB3\x02\uFBD5\x02" +
    "\uFD3F\x02\uFD52\x02\uFD91\x02\uFD94\x02\uFDC9\x02\uFDF2\x02\uFDFD\x02" +
    "\uFE02\x02\uFE11\x02\uFE22\x02\uFE31\x02\uFE35\x02\uFE36\x02\uFE4F\x02" +
    "\uFE51\x02\uFE72\x02\uFE76\x02\uFE78\x02\uFEFE\x02\uFF01\x02\uFF01\x02" +
    "\uFF12\x02\uFF1B\x02\uFF23\x02\uFF3C\x02\uFF41\x02\uFF41\x02\uFF43\x02" +
    "\uFF5C\x02\uFF68\x02\uFFC0\x02\uFFC4\x02\uFFC9\x02\uFFCC\x02\uFFD1\x02" +
    "\uFFD4\x02\uFFD9\x02\uFFDC\x02\uFFDE\x02\uFFFB\x02\uFFFD\x02\x02\x03\r" +
    "\x03\x0F\x03(\x03*\x03<\x03>\x03?\x03A\x03O\x03R\x03_\x03\x82\x03\xFC" +
    "\x03\u0142\x03\u0176\x03\u01FF\x03\u01FF\x03\u0282\x03\u029E\x03\u02A2" +
    "\x03\u02D2\x03\u02E2\x03\u02E2\x03\u0302\x03\u0321\x03\u0332\x03\u034C" +
    "\x03\u0352\x03\u037C\x03\u0382\x03\u039F\x03\u03A2\x03\u03C5\x03\u03CA" +
    "\x03\u03D1\x03\u03D3\x03\u03D7\x03\u0402\x03\u049F\x03\u04A2\x03\u04AB" +
    "\x03\u04B2\x03\u04D5\x03\u04DA\x03\u04FD\x03\u0502\x03\u0529\x03\u0532" +
    "\x03\u0565\x03\u0602\x03\u0738\x03\u0742\x03\u0757\x03\u0762\x03\u0769" +
    "\x03\u0802\x03\u0807\x03\u080A\x03\u080A\x03\u080C\x03\u0837\x03\u0839" +
    "\x03\u083A\x03\u083E\x03\u083E\x03\u0841\x03\u0857\x03\u0862\x03\u0878" +
    "\x03\u0882\x03\u08A0\x03\u08E2\x03\u08F4\x03\u08F6\x03\u08F7\x03\u0902" +
    "\x03\u0917\x03\u0922\x03\u093B\x03\u0982\x03\u09B9\x03\u09C0\x03\u09C1" +
    "\x03\u0A02\x03\u0A05\x03\u0A07\x03\u0A08\x03\u0A0E\x03\u0A15\x03\u0A17" +
    "\x03\u0A19\x03\u0A1B\x03\u0A35\x03\u0A3A\x03\u0A3C\x03\u0A41\x03\u0A41" +
    "\x03\u0A62\x03\u0A7E\x03\u0A82\x03\u0A9E\x03\u0AC2\x03\u0AC9\x03\u0ACB" +
    "\x03\u0AE8\x03\u0B02\x03\u0B37\x03\u0B42\x03\u0B57\x03\u0B62\x03\u0B74" +
    "\x03\u0B82\x03\u0B93\x03\u0C02\x03\u0C4A\x03\u0C82\x03\u0CB4\x03\u0CC2" +
    "\x03\u0CF4\x03\u1002\x03\u1048\x03\u1068\x03\u1071\x03\u1081\x03\u10BC" +
    "\x03\u10BF\x03\u10BF\x03\u10D2\x03\u10EA\x03\u10F2\x03\u10FB\x03\u1102" +
    "\x03\u1136\x03\u1138\x03\u1141\x03\u1152\x03\u1175\x03\u1178\x03\u1178" +
    "\x03\u1182\x03\u11C6\x03\u11CC\x03\u11CE\x03\u11D2\x03\u11DC\x03\u11DE" +
    "\x03\u11DE\x03\u1202\x03\u1213\x03\u1215\x03\u1239\x03\u1240\x03\u1240" +
    "\x03\u1282\x03\u1288\x03\u128A\x03\u128A\x03\u128C\x03\u128F\x03\u1291" +
    "\x03\u129F\x03\u12A1\x03\u12AA\x03\u12B2\x03\u12EC\x03\u12F2\x03\u12FB" +
    "\x03\u1302\x03\u1305\x03\u1307\x03\u130E\x03\u1311\x03\u1312\x03\u1315" +
    "\x03\u132A\x03\u132C\x03\u1332\x03\u1334\x03\u1335\x03\u1337\x03\u133B" +
    "\x03\u133E\x03\u1346\x03\u1349\x03\u134A\x03\u134D\x03\u134F\x03\u1352" +
    "\x03\u1352\x03\u1359\x03\u1359\x03\u135F\x03\u1365\x03\u1368\x03\u136E" +
    "\x03\u1372\x03\u1376\x03\u1402\x03\u144C\x03\u1452\x03\u145B\x03\u1482" +
    "\x03\u14C7\x03\u14C9\x03\u14C9\x03\u14D2\x03\u14DB\x03\u1582\x03\u15B7" +
    "\x03\u15BA\x03\u15C2\x03\u15DA\x03\u15DF\x03\u1602\x03\u1642\x03\u1646" +
    "\x03\u1646\x03\u1652\x03\u165B\x03\u1682\x03\u16B9\x03\u16C2\x03\u16CB" +
    "\x03\u1702\x03\u171B\x03\u171F\x03\u172D\x03\u1732\x03\u173B\x03\u18A2" +
    "\x03\u18EB\x03\u1901\x03\u1901\x03\u1AC2\x03\u1AFA\x03\u1C02\x03\u1C0A" +
    "\x03\u1C0C\x03\u1C38\x03\u1C3A\x03\u1C42\x03\u1C52\x03\u1C5B\x03\u1C74" +
    "\x03\u1C91\x03\u1C94\x03\u1CA9\x03\u1CAB\x03\u1CB8\x03\u2002\x03\u239B" +
    "\x03\u2402\x03\u2470\x03\u2482\x03\u2545\x03\u3002\x03\u3430\x03\u4402" +
    "\x03\u4648\x03\u6802\x03\u6A3A\x03\u6A42\x03\u6A60\x03\u6A62\x03\u6A6B" +
    "\x03\u6AD2\x03\u6AEF\x03\u6AF2\x03\u6AF6\x03\u6B02\x03\u6B38\x03\u6B42" +
    "\x03\u6B45\x03\u6B52\x03\u6B5B\x03\u6B65\x03\u6B79\x03\u6B7F\x03\u6B91" +
    "\x03\u6F02\x03\u6F46\x03\u6F52\x03\u6F80\x03\u6F91\x03\u6FA1\x03\u6FE2" +
    "\x03\u6FE2\x03\u7002\x03\u87EE\x03\u8802\x03\u8AF4\x03\uB002\x03\uB003" +
    "\x03\uBC02\x03\uBC6C\x03\uBC72\x03\uBC7E\x03\uBC82\x03\uBC8A\x03\uBC92" +
    "\x03\uBC9B\x03\uBC9F\x03\uBCA0\x03\uBCA2\x03\uBCA5\x03\uD167\x03\uD16B" +
    "\x03\uD16F\x03\uD184\x03\uD187\x03\uD18D\x03\uD1AC\x03\uD1AF\x03\uD244" +
    "\x03\uD246\x03\uD402\x03\uD456\x03\uD458\x03\uD49E\x03\uD4A0\x03\uD4A1" +
    "\x03\uD4A4\x03\uD4A4\x03\uD4A7\x03\uD4A8\x03\uD4AB\x03\uD4AE\x03\uD4B0" +
    "\x03\uD4BB\x03\uD4BD\x03\uD4BD\x03\uD4BF\x03\uD4C5\x03\uD4C7\x03\uD507" +
    "\x03\uD509\x03\uD50C\x03\uD50F\x03\uD516\x03\uD518\x03\uD51E\x03\uD520" +
    "\x03\uD53B\x03\uD53D\x03\uD540\x03\uD542\x03\uD546\x03\uD548\x03\uD548" +
    "\x03\uD54C\x03\uD552\x03\uD554\x03\uD6A7\x03\uD6AA\x03\uD6C2\x03\uD6C4" +
    "\x03\uD6DC\x03\uD6DE\x03\uD6FC\x03\uD6FE\x03\uD716\x03\uD718\x03\uD736" +
    "\x03\uD738\x03\uD750\x03\uD752\x03\uD770\x03\uD772\x03\uD78A\x03\uD78C" +
    "\x03\uD7AA\x03\uD7AC\x03\uD7C4\x03\uD7C6\x03\uD7CD\x03\uD7D0\x03\uD801" +
    "\x03\uDA02\x03\uDA38\x03\uDA3D\x03\uDA6E\x03\uDA77\x03\uDA77\x03\uDA86" +
    "\x03\uDA86\x03\uDA9D\x03\uDAA1\x03\uDAA3\x03\uDAB1\x03\uE002\x03\uE008" +
    "\x03\uE00A\x03\uE01A\x03\uE01D\x03\uE023\x03\uE025\x03\uE026\x03\uE028" +
    "\x03\uE02C\x03\uE802\x03\uE8C6\x03\uE8D2\x03\uE8D8\x03\uE902\x03\uE94C" +
    "\x03\uE952\x03\uE95B\x03\uEE02\x03\uEE05\x03\uEE07\x03\uEE21\x03\uEE23" +
    "\x03\uEE24\x03\uEE26\x03\uEE26\x03\uEE29\x03\uEE29\x03\uEE2B\x03\uEE34" +
    "\x03\uEE36\x03\uEE39\x03\uEE3B\x03\uEE3B\x03\uEE3D\x03\uEE3D\x03\uEE44" +
    "\x03\uEE44\x03\uEE49\x03\uEE49\x03\uEE4B\x03\uEE4B\x03\uEE4D\x03\uEE4D" +
    "\x03\uEE4F\x03\uEE51\x03\uEE53\x03\uEE54\x03\uEE56\x03\uEE56\x03\uEE59" +
    "\x03\uEE59\x03\uEE5B\x03\uEE5B\x03\uEE5D\x03\uEE5D\x03\uEE5F\x03\uEE5F" +
    "\x03\uEE61\x03\uEE61\x03\uEE63\x03\uEE64\x03\uEE66\x03\uEE66\x03\uEE69" +
    "\x03\uEE6C\x03\uEE6E\x03\uEE74\x03\uEE76\x03\uEE79\x03\uEE7B\x03\uEE7E" +
    "\x03\uEE80\x03\uEE80\x03\uEE82\x03\uEE8B\x03\uEE8D\x03\uEE9D\x03\uEEA3" +
    "\x03\uEEA5\x03\uEEA7\x03\uEEAB\x03\uEEAD\x03\uEEBD\x03\x02\x04\uA6D8\x04" +
    "\uA702\x04\uB736\x04\uB742\x04\uB81F\x04\uB822\x04\uCEA3\x04\uF802\x04" +
    "\uFA1F\x04\x03\x10\x03\x10\"\x10\x81\x10\u0102\x10\u01F1\x10\u0240\x02" +
    "C\x02\\\x02c\x02|\x02\xAC\x02\xAC\x02\xB7\x02\xB7\x02\xBC\x02\xBC\x02" +
    "\xC2\x02\xD8\x02\xDA\x02\xF8\x02\xFA\x02\u02C3\x02\u02C8\x02\u02D3\x02" +
    "\u02E2\x02\u02E6\x02\u02EE\x02\u02EE\x02\u02F0\x02\u02F0\x02\u0372\x02" +
    "\u0376\x02\u0378\x02\u0379\x02\u037C\x02\u037F\x02\u0381\x02\u0381\x02" +
    "\u0388\x02\u0388\x02\u038A\x02\u038C\x02\u038E\x02\u038E\x02\u0390\x02" +
    "\u03A3\x02\u03A5\x02\u03F7\x02\u03F9\x02\u0483\x02\u048C\x02\u0531\x02" +
    "\u0533\x02\u0558\x02\u055B\x02\u055B\x02\u0563\x02\u0589\x02\u05D2\x02" +
    "\u05EC\x02\u05F2\x02\u05F4\x02\u0622\x02\u064C\x02\u0670\x02\u0671\x02" +
    "\u0673\x02\u06D5\x02\u06D7\x02\u06D7\x02\u06E7\x02\u06E8\x02\u06F0\x02" +
    "\u06F1\x02\u06FC\x02\u06FE\x02\u0701\x02\u0701\x02\u0712\x02\u0712\x02" +
    "\u0714\x02\u0731\x02\u074F\x02\u07A7\x02\u07B3\x02\u07B3\x02\u07CC\x02" +
    "\u07EC\x02\u07F6\x02\u07F7\x02\u07FC\x02\u07FC\x02\u0802\x02\u0817\x02" +
    "\u081C\x02\u081C\x02\u0826\x02\u0826\x02\u082A\x02\u082A\x02\u0842\x02" +
    "\u085A\x02\u08A2\x02\u08B6\x02\u08B8\x02\u08BF\x02\u0906\x02\u093B\x02" +
    "\u093F\x02\u093F\x02\u0952\x02\u0952\x02\u095A\x02\u0963\x02\u0973\x02" +
    "\u0982\x02\u0987\x02\u098E\x02\u0991\x02\u0992\x02\u0995\x02\u09AA\x02" +
    "\u09AC\x02\u09B2\x02\u09B4\x02\u09B4\x02\u09B8\x02\u09BB\x02\u09BF\x02" +
    "\u09BF\x02\u09D0\x02\u09D0\x02\u09DE\x02\u09DF\x02\u09E1\x02\u09E3\x02" +
    "\u09F2\x02\u09F3\x02\u0A07\x02\u0A0C\x02\u0A11\x02\u0A12\x02\u0A15\x02" +
    "\u0A2A\x02\u0A2C\x02\u0A32\x02\u0A34\x02\u0A35\x02\u0A37\x02\u0A38\x02" +
    "\u0A3A\x02\u0A3B\x02\u0A5B\x02\u0A5E\x02\u0A60\x02\u0A60\x02\u0A74\x02" +
    "\u0A76\x02\u0A87\x02\u0A8F\x02\u0A91\x02\u0A93\x02\u0A95\x02\u0AAA\x02" +
    "\u0AAC\x02\u0AB2\x02\u0AB4\x02\u0AB5\x02\u0AB7\x02\u0ABB\x02\u0ABF\x02" +
    "\u0ABF\x02\u0AD2\x02\u0AD2\x02\u0AE2\x02\u0AE3\x02\u0AFB\x02\u0AFB\x02" +
    "\u0B07\x02\u0B0E\x02\u0B11\x02\u0B12\x02\u0B15\x02\u0B2A\x02\u0B2C\x02" +
    "\u0B32\x02\u0B34\x02\u0B35\x02\u0B37\x02\u0B3B\x02\u0B3F\x02\u0B3F\x02" +
    "\u0B5E\x02\u0B5F\x02\u0B61\x02\u0B63\x02\u0B73\x02\u0B73\x02\u0B85\x02" +
    "\u0B85\x02\u0B87\x02\u0B8C\x02\u0B90\x02\u0B92\x02\u0B94\x02\u0B97\x02" +
    "\u0B9B\x02\u0B9C\x02\u0B9E\x02\u0B9E\x02\u0BA0\x02\u0BA1\x02\u0BA5\x02" +
    "\u0BA6\x02\u0BAA\x02\u0BAC\x02\u0BB0\x02\u0BBB\x02\u0BD2\x02\u0BD2\x02" +
    "\u0C07\x02\u0C0E\x02\u0C10\x02\u0C12\x02\u0C14\x02\u0C2A\x02\u0C2C\x02" +
    "\u0C3B\x02\u0C3F\x02\u0C3F\x02\u0C5A\x02\u0C5C\x02\u0C62\x02\u0C63\x02" +
    "\u0C82\x02\u0C82\x02\u0C87\x02\u0C8E\x02\u0C90\x02\u0C92\x02\u0C94\x02" +
    "\u0CAA\x02\u0CAC\x02\u0CB5\x02\u0CB7\x02\u0CBB\x02\u0CBF\x02\u0CBF\x02" +
    "\u0CE0\x02\u0CE0\x02\u0CE2\x02\u0CE3\x02\u0CF3\x02\u0CF4\x02\u0D07\x02" +
    "\u0D0E\x02\u0D10\x02\u0D12\x02\u0D14\x02\u0D3C\x02\u0D3F\x02\u0D3F\x02" +
    "\u0D50\x02\u0D50\x02\u0D56\x02\u0D58\x02\u0D61\x02\u0D63\x02\u0D7C\x02" +
    "\u0D81\x02\u0D87\x02\u0D98\x02\u0D9C\x02\u0DB3\x02\u0DB5\x02\u0DBD\x02" +
    "\u0DBF\x02\u0DBF\x02\u0DC2\x02\u0DC8\x02\u0E03\x02\u0E32\x02\u0E34\x02" +
    "\u0E35\x02\u0E42\x02\u0E48\x02\u0E83\x02\u0E84\x02\u0E86\x02\u0E86\x02" +
    "\u0E89\x02\u0E8A\x02\u0E8C\x02\u0E8C\x02\u0E8F\x02\u0E8F\x02\u0E96\x02" +
    "\u0E99\x02\u0E9B\x02\u0EA1\x02\u0EA3\x02\u0EA5\x02\u0EA7\x02\u0EA7\x02" +
    "\u0EA9\x02\u0EA9\x02\u0EAC\x02\u0EAD\x02\u0EAF\x02\u0EB2\x02\u0EB4\x02" +
    "\u0EB5\x02\u0EBF\x02\u0EBF\x02\u0EC2\x02\u0EC6\x02\u0EC8\x02\u0EC8\x02" +
    "\u0EDE\x02\u0EE1\x02\u0F02\x02\u0F02\x02\u0F42\x02\u0F49\x02\u0F4B\x02" +
    "\u0F6E\x02\u0F8A\x02\u0F8E\x02\u1002\x02\u102C\x02\u1041\x02\u1041\x02" +
    "\u1052\x02\u1057\x02\u105C\x02\u105F\x02\u1063\x02\u1063\x02\u1067\x02" +
    "\u1068\x02\u1070\x02\u1072\x02\u1077\x02\u1083\x02\u1090\x02\u1090\x02" +
    "\u10A2\x02\u10C7\x02\u10C9\x02\u10C9\x02\u10CF\x02\u10CF\x02\u10D2\x02" +
    "\u10FC\x02\u10FE\x02\u124A\x02\u124C\x02\u124F\x02\u1252\x02\u1258\x02" +
    "\u125A\x02\u125A\x02\u125C\x02\u125F\x02\u1262\x02\u128A\x02\u128C\x02" +
    "\u128F\x02\u1292\x02\u12B2\x02\u12B4\x02\u12B7\x02\u12BA\x02\u12C0\x02" +
    "\u12C2\x02\u12C2\x02\u12C4\x02\u12C7\x02\u12CA\x02\u12D8\x02\u12DA\x02" +
    "\u1312\x02\u1314\x02\u1317\x02\u131A\x02\u135C\x02\u1382\x02\u1391\x02" +
    "\u13A2\x02\u13F7\x02\u13FA\x02\u13FF\x02\u1403\x02\u166E\x02\u1671\x02" +
    "\u1681\x02\u1683\x02\u169C\x02\u16A2\x02\u16EC\x02\u16F0\x02\u16FA\x02" +
    "\u1702\x02\u170E\x02\u1710\x02\u1713\x02\u1722\x02\u1733\x02\u1742\x02" +
    "\u1753\x02\u1762\x02\u176E\x02\u1770\x02\u1772\x02\u1782\x02\u17B5\x02" +
    "\u17D9\x02\u17D9\x02\u17DE\x02\u17DE\x02\u1822\x02\u1879\x02\u1882\x02" +
    "\u1886\x02\u1889\x02\u18AA\x02\u18AC\x02\u18AC\x02\u18B2\x02\u18F7\x02" +
    "\u1902\x02\u1920\x02\u1952\x02\u196F\x02\u1972\x02\u1976\x02\u1982\x02" +
    "\u19AD\x02\u19B2\x02\u19CB\x02\u1A02\x02\u1A18\x02\u1A22\x02\u1A56\x02" +
    "\u1AA9\x02\u1AA9\x02\u1B07\x02\u1B35\x02\u1B47\x02\u1B4D\x02\u1B85\x02" +
    "\u1BA2\x02\u1BB0\x02\u1BB1\x02\u1BBC\x02\u1BE7\x02\u1C02\x02\u1C25\x02" +
    "\u1C4F\x02\u1C51\x02\u1C5C\x02\u1C7F\x02\u1C82\x02\u1C8A\x02\u1CEB\x02" +
    "\u1CEE\x02\u1CF0\x02\u1CF3\x02\u1CF7\x02\u1CF8\x02\u1D02\x02\u1DC1\x02" +
    "\u1E02\x02\u1F17\x02\u1F1A\x02\u1F1F\x02\u1F22\x02\u1F47\x02\u1F4A\x02" +
    "\u1F4F\x02\u1F52\x02\u1F59\x02\u1F5B\x02\u1F5B\x02\u1F5D\x02\u1F5D\x02" +
    "\u1F5F\x02\u1F5F\x02\u1F61\x02\u1F7F\x02\u1F82\x02\u1FB6\x02\u1FB8\x02" +
    "\u1FBE\x02\u1FC0\x02\u1FC0\x02\u1FC4\x02\u1FC6\x02\u1FC8\x02\u1FCE\x02" +
    "\u1FD2\x02\u1FD5\x02\u1FD8\x02\u1FDD\x02\u1FE2\x02\u1FEE\x02\u1FF4\x02" +
    "\u1FF6\x02\u1FF8\x02\u1FFE\x02\u2073\x02\u2073\x02\u2081\x02\u2081\x02" +
    "\u2092\x02\u209E\x02\u2104\x02\u2104\x02\u2109\x02\u2109\x02\u210C\x02" +
    "\u2115\x02\u2117\x02\u2117\x02\u211B\x02\u211F\x02\u2126\x02\u2126\x02" +
    "\u2128\x02\u2128\x02\u212A\x02\u212A\x02\u212C\x02\u212F\x02\u2131\x02" +
    "\u213B\x02\u213E\x02\u2141\x02\u2147\x02\u214B\x02\u2150\x02\u2150\x02" +
    "\u2162\x02\u218A\x02\u2C02\x02\u2C30\x02\u2C32\x02\u2C60\x02\u2C62\x02" +
    "\u2CE6\x02\u2CED\x02\u2CF0\x02\u2CF4\x02\u2CF5\x02\u2D02\x02\u2D27\x02" +
    "\u2D29\x02\u2D29\x02\u2D2F\x02\u2D2F\x02\u2D32\x02\u2D69\x02\u2D71\x02" +
    "\u2D71\x02\u2D82\x02\u2D98\x02\u2DA2\x02\u2DA8\x02\u2DAA\x02\u2DB0\x02" +
    "\u2DB2\x02\u2DB8\x02\u2DBA\x02\u2DC0\x02\u2DC2\x02\u2DC8\x02\u2DCA\x02" +
    "\u2DD0\x02\u2DD2\x02\u2DD8\x02\u2DDA\x02\u2DE0\x02\u2E31\x02\u2E31\x02" +
    "\u3007\x02\u3009\x02\u3023\x02\u302B\x02\u3033\x02\u3037\x02\u303A\x02" +
    "\u303E\x02\u3043\x02\u3098\x02\u309F\x02\u30A1\x02\u30A3\x02\u30FC\x02" +
    "\u30FE\x02\u3101\x02\u3107\x02\u312F\x02\u3133\x02\u3190\x02\u31A2\x02" +
    "\u31BC\x02\u31F2\x02\u3201\x02\u3402\x02\u4DB7\x02\u4E02\x02\u9FD7\x02" +
    "\uA002\x02\uA48E\x02\uA4D2\x02\uA4FF\x02\uA502\x02\uA60E\x02\uA612\x02" +
    "\uA621\x02\uA62C\x02\uA62D\x02\uA642\x02\uA670\x02\uA681\x02\uA69F\x02" +
    "\uA6A2\x02\uA6F1\x02\uA719\x02\uA721\x02\uA724\x02\uA78A\x02\uA78D\x02" +
    "\uA7B0\x02\uA7B2\x02\uA7B9\x02\uA7F9\x02\uA803\x02\uA805\x02\uA807\x02" +
    "\uA809\x02\uA80C\x02\uA80E\x02\uA824\x02\uA842\x02\uA875\x02\uA884\x02" +
    "\uA8B5\x02\uA8F4\x02\uA8F9\x02\uA8FD\x02\uA8FD\x02\uA8FF\x02\uA8FF\x02" +
    "\uA90C\x02\uA927\x02\uA932\x02\uA948\x02\uA962\x02\uA97E\x02\uA986\x02" +
    "\uA9B4\x02\uA9D1\x02\uA9D1\x02\uA9E2\x02\uA9E6\x02\uA9E8\x02\uA9F1\x02" +
    "\uA9FC\x02\uAA00\x02\uAA02\x02\uAA2A\x02\uAA42\x02\uAA44\x02\uAA46\x02" +
    "\uAA4D\x02\uAA62\x02\uAA78\x02\uAA7C\x02\uAA7C\x02\uAA80\x02\uAAB1\x02" +
    "\uAAB3\x02\uAAB3\x02\uAAB7\x02\uAAB8\x02\uAABB\x02\uAABF\x02\uAAC2\x02" +
    "\uAAC2\x02\uAAC4\x02\uAAC4\x02\uAADD\x02\uAADF\x02\uAAE2\x02\uAAEC\x02" +
    "\uAAF4\x02\uAAF6\x02\uAB03\x02\uAB08\x02\uAB0B\x02\uAB10\x02\uAB13\x02" +
    "\uAB18\x02\uAB22\x02\uAB28\x02\uAB2A\x02\uAB30\x02\uAB32\x02\uAB5C\x02" +
    "\uAB5E\x02\uAB67\x02\uAB72\x02\uABE4\x02\uAC02\x02\uD7A5\x02\uD7B2\x02" +
    "\uD7C8\x02\uD7CD\x02\uD7FD\x02\uF902\x02\uFA6F\x02\uFA72\x02\uFADB\x02" +
    "\uFB02\x02\uFB08\x02\uFB15\x02\uFB19\x02\uFB1F\x02\uFB1F\x02\uFB21\x02" +
    "\uFB2A\x02\uFB2C\x02\uFB38\x02\uFB3A\x02\uFB3E\x02\uFB40\x02\uFB40\x02" +
    "\uFB42\x02\uFB43\x02\uFB45\x02\uFB46\x02\uFB48\x02\uFBB3\x02\uFBD5\x02" +
    "\uFD3F\x02\uFD52\x02\uFD91\x02\uFD94\x02\uFDC9\x02\uFDF2\x02\uFDFD\x02" +
    "\uFE72\x02\uFE76\x02\uFE78\x02\uFEFE\x02\uFF23\x02\uFF3C\x02\uFF43\x02" +
    "\uFF5C\x02\uFF68\x02\uFFC0\x02\uFFC4\x02\uFFC9\x02\uFFCC\x02\uFFD1\x02" +
    "\uFFD4\x02\uFFD9\x02\uFFDC\x02\uFFDE\x02\x02\x03\r\x03\x0F\x03(\x03*\x03" +
    "<\x03>\x03?\x03A\x03O\x03R\x03_\x03\x82\x03\xFC\x03\u0142\x03\u0176\x03" +
    "\u0282\x03\u029E\x03\u02A2\x03\u02D2\x03\u0302\x03\u0321\x03\u0332\x03" +
    "\u034C\x03\u0352\x03\u0377\x03\u0382\x03\u039F\x03\u03A2\x03\u03C5\x03" +
    "\u03CA\x03\u03D1\x03\u03D3\x03\u03D7\x03\u0402\x03\u049F\x03\u04B2\x03" +
    "\u04D5\x03\u04DA\x03\u04FD\x03\u0502\x03\u0529\x03\u0532\x03\u0565\x03" +
    "\u0602\x03\u0738\x03\u0742\x03\u0757\x03\u0762\x03\u0769\x03\u0802\x03" +
    "\u0807\x03\u080A\x03\u080A\x03\u080C\x03\u0837\x03\u0839\x03\u083A\x03" +
    "\u083E\x03\u083E\x03\u0841\x03\u0857\x03\u0862\x03\u0878\x03\u0882\x03" +
    "\u08A0\x03\u08E2\x03\u08F4\x03\u08F6\x03\u08F7\x03\u0902\x03\u0917\x03" +
    "\u0922\x03\u093B\x03\u0982\x03\u09B9\x03\u09C0\x03\u09C1\x03\u0A02\x03" +
    "\u0A02\x03\u0A12\x03\u0A15\x03\u0A17\x03\u0A19\x03\u0A1B\x03\u0A35\x03" +
    "\u0A62\x03\u0A7E\x03\u0A82\x03\u0A9E\x03\u0AC2\x03\u0AC9\x03\u0ACB\x03" +
    "\u0AE6\x03\u0B02\x03\u0B37\x03\u0B42\x03\u0B57\x03\u0B62\x03\u0B74\x03" +
    "\u0B82\x03\u0B93\x03\u0C02\x03\u0C4A\x03\u0C82\x03\u0CB4\x03\u0CC2\x03" +
    "\u0CF4\x03\u1005\x03\u1039\x03\u1085\x03\u10B1\x03\u10D2\x03\u10EA\x03" +
    "\u1105\x03\u1128\x03\u1152\x03\u1174\x03\u1178\x03\u1178\x03\u1185\x03" +
    "\u11B4\x03\u11C3\x03\u11C6\x03\u11DC\x03\u11DC\x03\u11DE\x03\u11DE\x03" +
    "\u1202\x03\u1213\x03\u1215\x03\u122D\x03\u1282\x03\u1288\x03\u128A\x03" +
    "\u128A\x03\u128C\x03\u128F\x03\u1291\x03\u129F\x03\u12A1\x03\u12AA\x03" +
    "\u12B2\x03\u12E0\x03\u1307\x03\u130E\x03\u1311\x03\u1312\x03\u1315\x03" +
    "\u132A\x03\u132C\x03\u1332\x03\u1334\x03\u1335\x03\u1337\x03\u133B\x03" +
    "\u133F\x03\u133F\x03\u1352\x03\u1352\x03\u135F\x03\u1363\x03\u1402\x03" +
    "\u1436\x03\u1449\x03\u144C\x03\u1482\x03\u14B1\x03\u14C6\x03\u14C7\x03" +
    "\u14C9\x03\u14C9\x03\u1582\x03\u15B0\x03\u15DA\x03\u15DD\x03\u1602\x03" +
    "\u1631\x03\u1646\x03\u1646\x03\u1682\x03\u16AC\x03\u1702\x03\u171B\x03" +
    "\u18A2\x03\u18E1\x03\u1901\x03\u1901\x03\u1AC2\x03\u1AFA\x03\u1C02\x03" +
    "\u1C0A\x03\u1C0C\x03\u1C30\x03\u1C42\x03\u1C42\x03\u1C74\x03\u1C91\x03" +
    "\u2002\x03\u239B\x03\u2402\x03\u2470\x03\u2482\x03\u2545\x03\u3002\x03" +
    "\u3430\x03\u4402\x03\u4648\x03\u6802\x03\u6A3A\x03\u6A42\x03\u6A60\x03" +
    "\u6AD2\x03\u6AEF\x03\u6B02\x03\u6B31\x03\u6B42\x03\u6B45\x03\u6B65\x03" +
    "\u6B79\x03\u6B7F\x03\u6B91\x03\u6F02\x03\u6F46\x03\u6F52\x03\u6F52\x03" +
    "\u6F95\x03\u6FA1\x03\u6FE2\x03\u6FE2\x03\u7002\x03\u87EE\x03\u8802\x03" +
    "\u8AF4\x03\uB002\x03\uB003\x03\uBC02\x03\uBC6C\x03\uBC72\x03\uBC7E\x03" +
    "\uBC82\x03\uBC8A\x03\uBC92\x03\uBC9B\x03\uD402\x03\uD456\x03\uD458\x03" +
    "\uD49E\x03\uD4A0\x03\uD4A1\x03\uD4A4\x03\uD4A4\x03\uD4A7\x03\uD4A8\x03" +
    "\uD4AB\x03\uD4AE\x03\uD4B0\x03\uD4BB\x03\uD4BD\x03\uD4BD\x03\uD4BF\x03" +
    "\uD4C5\x03\uD4C7\x03\uD507\x03\uD509\x03\uD50C\x03\uD50F\x03\uD516\x03" +
    "\uD518\x03\uD51E\x03\uD520\x03\uD53B\x03\uD53D\x03\uD540\x03\uD542\x03" +
    "\uD546\x03\uD548\x03\uD548";
XPathLexer._serializedATNSegment1 = "\x03\uD54C\x03\uD552\x03\uD554\x03\uD6A7\x03\uD6AA\x03\uD6C2\x03\uD6C4" +
    "\x03\uD6DC\x03\uD6DE\x03\uD6FC\x03\uD6FE\x03\uD716\x03\uD718\x03\uD736" +
    "\x03\uD738\x03\uD750\x03\uD752\x03\uD770\x03\uD772\x03\uD78A\x03\uD78C" +
    "\x03\uD7AA\x03\uD7AC\x03\uD7C4\x03\uD7C6\x03\uD7CD\x03\uE802\x03\uE8C6" +
    "\x03\uE902\x03\uE945\x03\uEE02\x03\uEE05\x03\uEE07\x03\uEE21\x03\uEE23" +
    "\x03\uEE24\x03\uEE26\x03\uEE26\x03\uEE29\x03\uEE29\x03\uEE2B\x03\uEE34" +
    "\x03\uEE36\x03\uEE39\x03\uEE3B\x03\uEE3B\x03\uEE3D\x03\uEE3D\x03\uEE44" +
    "\x03\uEE44\x03\uEE49\x03\uEE49\x03\uEE4B\x03\uEE4B\x03\uEE4D\x03\uEE4D" +
    "\x03\uEE4F\x03\uEE51\x03\uEE53\x03\uEE54\x03\uEE56\x03\uEE56\x03\uEE59" +
    "\x03\uEE59\x03\uEE5B\x03\uEE5B\x03\uEE5D\x03\uEE5D\x03\uEE5F\x03\uEE5F" +
    "\x03\uEE61\x03\uEE61\x03\uEE63\x03\uEE64\x03\uEE66\x03\uEE66\x03\uEE69" +
    "\x03\uEE6C\x03\uEE6E\x03\uEE74\x03\uEE76\x03\uEE79\x03\uEE7B\x03\uEE7E" +
    "\x03\uEE80\x03\uEE80\x03\uEE82\x03\uEE8B\x03\uEE8D\x03\uEE9D\x03\uEEA3" +
    "\x03\uEEA5\x03\uEEA7\x03\uEEAB\x03\uEEAD\x03\uEEBD\x03\x02\x04\uA6D8\x04" +
    "\uA702\x04\uB736\x04\uB742\x04\uB81F\x04\uB822\x04\uCEA3\x04\uF802\x04" +
    "\uFA1F\x041\x02\x03\x03\x02\x02\x02\x02\x05\x03\x02\x02\x02\x02\x07\x03" +
    "\x02\x02\x02\x02\t\x03\x02\x02\x02\x02\v\x03\x02\x02\x02\x02\x11\x03\x02" +
    "\x02\x02\x03\x13\x03\x02\x02\x02\x05\x16\x03\x02\x02\x02\x07\x18\x03\x02" +
    "\x02\x02\t\x1A\x03\x02\x02\x02\v\x1C\x03\x02\x02\x02\r%\x03\x02\x02\x02" +
    "\x0F\'\x03\x02\x02\x02\x11)\x03\x02\x02\x02\x13\x14\x071\x02\x02\x14\x15" +
    "\x071\x02\x02\x15\x04\x03\x02\x02\x02\x16\x17\x071\x02\x02\x17\x06\x03" +
    "\x02\x02\x02\x18\x19\x07,\x02\x02\x19\b\x03\x02\x02\x02\x1A\x1B\x07#\x02" +
    "\x02\x1B\n\x03\x02\x02\x02\x1C \x05\x0F\b\x02\x1D\x1F\x05\r\x07\x02\x1E" +
    "\x1D\x03\x02\x02\x02\x1F\"\x03\x02\x02\x02 \x1E\x03\x02\x02\x02 !\x03" +
    "\x02\x02\x02!#\x03\x02\x02\x02\" \x03\x02\x02\x02#$\b\x06\x02\x02$\f\x03" +
    "\x02\x02\x02%&\t\x02\x02\x02&\x0E\x03\x02\x02\x02\'(\t\x03\x02\x02(\x10" +
    "\x03\x02\x02\x02)-\x07)\x02\x02*,\v\x02\x02\x02+*\x03\x02\x02\x02,/\x03" +
    "\x02\x02\x02-.\x03\x02\x02\x02-+\x03\x02\x02\x02.0\x03\x02\x02\x02/-\x03" +
    "\x02\x02\x0201\x07)\x02\x021\x12\x03\x02\x02\x02\x05\x02 -\x03\x03\x06" +
    "\x02";
XPathLexer._serializedATN = Utils.join([
    XPathLexer._serializedATNSegment0,
    XPathLexer._serializedATNSegment1,
], "");
//# sourceMappingURL=XPathLexer.js.map