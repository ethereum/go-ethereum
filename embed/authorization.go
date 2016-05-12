package embed

// AuthorizationHTML contains the html file content to support transaction authorization via popup without the need for CORS
const AuthorizationHTML = `
<!DOCTYPE html>
<html>
	<head>
		<title>Ethereum Authorization</title>
	</head>
	<script>
		//https://github.com/alexvandesande/blockies
		!function(){function r(r){for(var t=0;t<l.length;t++)l[t]=0;for(var t=0;t<r.length;t++)l[t%4]=(l[t%4]<<5)-l[t%4]+r.charCodeAt(t)}function t(){var r=l[0]^l[0]<<11;return l[0]=l[1],l[1]=l[2],l[2]=l[3],l[3]=l[3]^l[3]>>19^r^r>>8,(l[3]>>>0)/(1<<31>>>0)}function e(){var r=Math.floor(360*t()),e=60*t()+40+"%",o=25*(t()+t()+t()+t())+"%",n="hsl("+r+","+e+","+o+")";return n}function o(r){for(var e=r,o=r,n=Math.ceil(e/2),a=e-n,l=[],c=0;o>c;c++){for(var f=[],h=0;n>h;h++)f[h]=Math.floor(2.3*t());var i=f.slice(0,a);i.reverse(),f=f.concat(i);for(var v=0;v<f.length;v++)l.push(f[v])}return l}function n(r,t,e,o,n){var a=document.createElement("canvas"),l=Math.sqrt(r.length);a.width=a.height=l*e;var c=a.getContext("2d");c.fillStyle=o,c.fillRect(0,0,a.width,a.height),c.fillStyle=t;for(var f=0;f<r.length;f++){var h=Math.floor(f/l),i=f%l;c.fillStyle=1==r[f]?t:n,r[f]&&c.fillRect(i*e,h*e,e,e)}return a}function a(t){t=t||{};var a=t.size||8,l=t.scale||4,c=t.seed||Math.floor(Math.random()*Math.pow(10,16)).toString(16);r(c);var f=t.color||e(),h=t.bgcolor||e(),i=t.spotcolor||e(),v=o(a),u=n(v,f,l,h,i);return u}var l=new Array(4);window.blockies={create:a}}();
		
		/* bignumber.js v2.3.0 https://github.com/MikeMcl/bignumber.js/LICENCE */
		!function(e){"use strict";function n(e){function E(e,n){var t,r,i,o,u,s,f=this;if(!(f instanceof E))return j&&L(26,"constructor call without new",e),new E(e,n);if(null!=n&&H(n,2,64,M,"base")){if(n=0|n,s=e+"",10==n)return f=new E(e instanceof E?e:s),U(f,P+f.e+1,k);if((o="number"==typeof e)&&0*e!=0||!new RegExp("^-?"+(t="["+N.slice(0,n)+"]+")+"(?:\\."+t+")?$",37>n?"i":"").test(s))return h(f,s,o,n);o?(f.s=0>1/e?(s=s.slice(1),-1):1,j&&s.replace(/^0\.0*|\./,"").length>15&&L(M,v,e),o=!1):f.s=45===s.charCodeAt(0)?(s=s.slice(1),-1):1,s=D(s,10,n,f.s)}else{if(e instanceof E)return f.s=e.s,f.e=e.e,f.c=(e=e.c)?e.slice():e,void(M=0);if((o="number"==typeof e)&&0*e==0){if(f.s=0>1/e?(e=-e,-1):1,e===~~e){for(r=0,i=e;i>=10;i/=10,r++);return f.e=r,f.c=[e],void(M=0)}s=e+""}else{if(!g.test(s=e+""))return h(f,s,o);f.s=45===s.charCodeAt(0)?(s=s.slice(1),-1):1}}for((r=s.indexOf("."))>-1&&(s=s.replace(".","")),(i=s.search(/e/i))>0?(0>r&&(r=i),r+=+s.slice(i+1),s=s.substring(0,i)):0>r&&(r=s.length),i=0;48===s.charCodeAt(i);i++);for(u=s.length;48===s.charCodeAt(--u););if(s=s.slice(i,u+1))if(u=s.length,o&&j&&u>15&&(e>y||e!==d(e))&&L(M,v,f.s*e),r=r-i-1,r>z)f.c=f.e=null;else if(G>r)f.c=[f.e=0];else{if(f.e=r,f.c=[],i=(r+1)%O,0>r&&(i+=O),u>i){for(i&&f.c.push(+s.slice(0,i)),u-=O;u>i;)f.c.push(+s.slice(i,i+=O));s=s.slice(i),i=O-s.length}else i-=u;for(;i--;s+="0");f.c.push(+s)}else f.c=[f.e=0];M=0}function D(e,n,t,i){var o,u,f,c,a,h,g,p=e.indexOf("."),d=P,m=k;for(37>t&&(e=e.toLowerCase()),p>=0&&(f=J,J=0,e=e.replace(".",""),g=new E(t),a=g.pow(e.length-p),J=f,g.c=s(l(r(a.c),a.e),10,n),g.e=g.c.length),h=s(e,t,n),u=f=h.length;0==h[--f];h.pop());if(!h[0])return"0";if(0>p?--u:(a.c=h,a.e=u,a.s=i,a=C(a,g,d,m,n),h=a.c,c=a.r,u=a.e),o=u+d+1,p=h[o],f=n/2,c=c||0>o||null!=h[o+1],c=4>m?(null!=p||c)&&(0==m||m==(a.s<0?3:2)):p>f||p==f&&(4==m||c||6==m&&1&h[o-1]||m==(a.s<0?8:7)),1>o||!h[0])e=c?l("1",-d):"0";else{if(h.length=o,c)for(--n;++h[--o]>n;)h[o]=0,o||(++u,h.unshift(1));for(f=h.length;!h[--f];);for(p=0,e="";f>=p;e+=N.charAt(h[p++]));e=l(e,u)}return e}function F(e,n,t,i){var o,u,s,c,a;if(t=null!=t&&H(t,0,8,i,w)?0|t:k,!e.c)return e.toString();if(o=e.c[0],s=e.e,null==n)a=r(e.c),a=19==i||24==i&&B>=s?f(a,s):l(a,s);else if(e=U(new E(e),n,t),u=e.e,a=r(e.c),c=a.length,19==i||24==i&&(u>=n||B>=u)){for(;n>c;a+="0",c++);a=f(a,u)}else if(n-=s,a=l(a,u),u+1>c){if(--n>0)for(a+=".";n--;a+="0");}else if(n+=u-c,n>0)for(u+1==c&&(a+=".");n--;a+="0");return e.s<0&&o?"-"+a:a}function _(e,n){var t,r,i=0;for(u(e[0])&&(e=e[0]),t=new E(e[0]);++i<e.length;){if(r=new E(e[i]),!r.s){t=r;break}n.call(t,r)&&(t=r)}return t}function x(e,n,t,r,i){return(n>e||e>t||e!=c(e))&&L(r,(i||"decimal places")+(n>e||e>t?" out of range":" not an integer"),e),!0}function I(e,n,t){for(var r=1,i=n.length;!n[--i];n.pop());for(i=n[0];i>=10;i/=10,r++);return(t=r+t*O-1)>z?e.c=e.e=null:G>t?e.c=[e.e=0]:(e.e=t,e.c=n),e}function L(e,n,t){var r=new Error(["new BigNumber","cmp","config","div","divToInt","eq","gt","gte","lt","lte","minus","mod","plus","precision","random","round","shift","times","toDigits","toExponential","toFixed","toFormat","toFraction","pow","toPrecision","toString","BigNumber"][e]+"() "+n+": "+t);throw r.name="BigNumber Error",M=0,r}function U(e,n,t,r){var i,o,u,s,f,l,c,a=e.c,h=S;if(a){e:{for(i=1,s=a[0];s>=10;s/=10,i++);if(o=n-i,0>o)o+=O,u=n,f=a[l=0],c=f/h[i-u-1]%10|0;else if(l=p((o+1)/O),l>=a.length){if(!r)break e;for(;a.length<=l;a.push(0));f=c=0,i=1,o%=O,u=o-O+1}else{for(f=s=a[l],i=1;s>=10;s/=10,i++);o%=O,u=o-O+i,c=0>u?0:f/h[i-u-1]%10|0}if(r=r||0>n||null!=a[l+1]||(0>u?f:f%h[i-u-1]),r=4>t?(c||r)&&(0==t||t==(e.s<0?3:2)):c>5||5==c&&(4==t||r||6==t&&(o>0?u>0?f/h[i-u]:0:a[l-1])%10&1||t==(e.s<0?8:7)),1>n||!a[0])return a.length=0,r?(n-=e.e+1,a[0]=h[(O-n%O)%O],e.e=-n||0):a[0]=e.e=0,e;if(0==o?(a.length=l,s=1,l--):(a.length=l+1,s=h[O-o],a[l]=u>0?d(f/h[i-u]%h[u])*s:0),r)for(;;){if(0==l){for(o=1,u=a[0];u>=10;u/=10,o++);for(u=a[0]+=s,s=1;u>=10;u/=10,s++);o!=s&&(e.e++,a[0]==b&&(a[0]=1));break}if(a[l]+=s,a[l]!=b)break;a[l--]=0,s=1}for(o=a.length;0===a[--o];a.pop());}e.e>z?e.c=e.e=null:e.e<G&&(e.c=[e.e=0])}return e}var C,M=0,T=E.prototype,q=new E(1),P=20,k=4,B=-7,$=21,G=-1e7,z=1e7,j=!0,H=x,V=!1,W=1,J=100,X={decimalSeparator:".",groupSeparator:",",groupSize:3,secondaryGroupSize:0,fractionGroupSeparator:" ",fractionGroupSize:0};return E.another=n,E.ROUND_UP=0,E.ROUND_DOWN=1,E.ROUND_CEIL=2,E.ROUND_FLOOR=3,E.ROUND_HALF_UP=4,E.ROUND_HALF_DOWN=5,E.ROUND_HALF_EVEN=6,E.ROUND_HALF_CEIL=7,E.ROUND_HALF_FLOOR=8,E.EUCLID=9,E.config=function(){var e,n,t=0,r={},i=arguments,s=i[0],f=s&&"object"==typeof s?function(){return s.hasOwnProperty(n)?null!=(e=s[n]):void 0}:function(){return i.length>t?null!=(e=i[t++]):void 0};return f(n="DECIMAL_PLACES")&&H(e,0,A,2,n)&&(P=0|e),r[n]=P,f(n="ROUNDING_MODE")&&H(e,0,8,2,n)&&(k=0|e),r[n]=k,f(n="EXPONENTIAL_AT")&&(u(e)?H(e[0],-A,0,2,n)&&H(e[1],0,A,2,n)&&(B=0|e[0],$=0|e[1]):H(e,-A,A,2,n)&&(B=-($=0|(0>e?-e:e)))),r[n]=[B,$],f(n="RANGE")&&(u(e)?H(e[0],-A,-1,2,n)&&H(e[1],1,A,2,n)&&(G=0|e[0],z=0|e[1]):H(e,-A,A,2,n)&&(0|e?G=-(z=0|(0>e?-e:e)):j&&L(2,n+" cannot be zero",e))),r[n]=[G,z],f(n="ERRORS")&&(e===!!e||1===e||0===e?(M=0,H=(j=!!e)?x:o):j&&L(2,n+m,e)),r[n]=j,f(n="CRYPTO")&&(e===!!e||1===e||0===e?(V=!(!e||!a),e&&!V&&j&&L(2,"crypto unavailable",a)):j&&L(2,n+m,e)),r[n]=V,f(n="MODULO_MODE")&&H(e,0,9,2,n)&&(W=0|e),r[n]=W,f(n="POW_PRECISION")&&H(e,0,A,2,n)&&(J=0|e),r[n]=J,f(n="FORMAT")&&("object"==typeof e?X=e:j&&L(2,n+" not an object",e)),r[n]=X,r},E.max=function(){return _(arguments,T.lt)},E.min=function(){return _(arguments,T.gt)},E.random=function(){var e=9007199254740992,n=Math.random()*e&2097151?function(){return d(Math.random()*e)}:function(){return 8388608*(1073741824*Math.random()|0)+(8388608*Math.random()|0)};return function(e){var t,r,i,o,u,s=0,f=[],l=new E(q);if(e=null!=e&&H(e,0,A,14)?0|e:P,o=p(e/O),V)if(a&&a.getRandomValues){for(t=a.getRandomValues(new Uint32Array(o*=2));o>s;)u=131072*t[s]+(t[s+1]>>>11),u>=9e15?(r=a.getRandomValues(new Uint32Array(2)),t[s]=r[0],t[s+1]=r[1]):(f.push(u%1e14),s+=2);s=o/2}else if(a&&a.randomBytes){for(t=a.randomBytes(o*=7);o>s;)u=281474976710656*(31&t[s])+1099511627776*t[s+1]+4294967296*t[s+2]+16777216*t[s+3]+(t[s+4]<<16)+(t[s+5]<<8)+t[s+6],u>=9e15?a.randomBytes(7).copy(t,s):(f.push(u%1e14),s+=7);s=o/7}else j&&L(14,"crypto unavailable",a);if(!s)for(;o>s;)u=n(),9e15>u&&(f[s++]=u%1e14);for(o=f[--s],e%=O,o&&e&&(u=S[O-e],f[s]=d(o/u)*u);0===f[s];f.pop(),s--);if(0>s)f=[i=0];else{for(i=-1;0===f[0];f.shift(),i-=O);for(s=1,u=f[0];u>=10;u/=10,s++);O>s&&(i-=O-s)}return l.e=i,l.c=f,l}}(),C=function(){function e(e,n,t){var r,i,o,u,s=0,f=e.length,l=n%R,c=n/R|0;for(e=e.slice();f--;)o=e[f]%R,u=e[f]/R|0,r=c*o+u*l,i=l*o+r%R*R+s,s=(i/t|0)+(r/R|0)+c*u,e[f]=i%t;return s&&e.unshift(s),e}function n(e,n,t,r){var i,o;if(t!=r)o=t>r?1:-1;else for(i=o=0;t>i;i++)if(e[i]!=n[i]){o=e[i]>n[i]?1:-1;break}return o}function r(e,n,t,r){for(var i=0;t--;)e[t]-=i,i=e[t]<n[t]?1:0,e[t]=i*r+e[t]-n[t];for(;!e[0]&&e.length>1;e.shift());}return function(i,o,u,s,f){var l,c,a,h,g,p,m,w,v,N,y,S,R,A,D,F,_,x=i.s==o.s?1:-1,I=i.c,L=o.c;if(!(I&&I[0]&&L&&L[0]))return new E(i.s&&o.s&&(I?!L||I[0]!=L[0]:L)?I&&0==I[0]||!L?0*x:x/0:NaN);for(w=new E(x),v=w.c=[],c=i.e-o.e,x=u+c+1,f||(f=b,c=t(i.e/O)-t(o.e/O),x=x/O|0),a=0;L[a]==(I[a]||0);a++);if(L[a]>(I[a]||0)&&c--,0>x)v.push(1),h=!0;else{for(A=I.length,F=L.length,a=0,x+=2,g=d(f/(L[0]+1)),g>1&&(L=e(L,g,f),I=e(I,g,f),F=L.length,A=I.length),R=F,N=I.slice(0,F),y=N.length;F>y;N[y++]=0);_=L.slice(),_.unshift(0),D=L[0],L[1]>=f/2&&D++;do{if(g=0,l=n(L,N,F,y),0>l){if(S=N[0],F!=y&&(S=S*f+(N[1]||0)),g=d(S/D),g>1)for(g>=f&&(g=f-1),p=e(L,g,f),m=p.length,y=N.length;1==n(p,N,m,y);)g--,r(p,m>F?_:L,m,f),m=p.length,l=1;else 0==g&&(l=g=1),p=L.slice(),m=p.length;if(y>m&&p.unshift(0),r(N,p,y,f),y=N.length,-1==l)for(;n(L,N,F,y)<1;)g++,r(N,y>F?_:L,y,f),y=N.length}else 0===l&&(g++,N=[0]);v[a++]=g,N[0]?N[y++]=I[R]||0:(N=[I[R]],y=1)}while((R++<A||null!=N[0])&&x--);h=null!=N[0],v[0]||v.shift()}if(f==b){for(a=1,x=v[0];x>=10;x/=10,a++);U(w,u+(w.e=a+c*O-1)+1,s,h)}else w.e=c,w.r=+h;return w}}(),h=function(){var e=/^(-?)0([xbo])(?=\w[\w.]*$)/i,n=/^([^.]+)\.$/,t=/^\.([^.]+)$/,r=/^-?(Infinity|NaN)$/,i=/^\s*\+(?=[\w.])|^\s+|\s+$/g;return function(o,u,s,f){var l,c=s?u:u.replace(i,"");if(r.test(c))o.s=isNaN(c)?null:0>c?-1:1;else{if(!s&&(c=c.replace(e,function(e,n,t){return l="x"==(t=t.toLowerCase())?16:"b"==t?2:8,f&&f!=l?e:n}),f&&(l=f,c=c.replace(n,"$1").replace(t,"0.$1")),u!=c))return new E(c,l);j&&L(M,"not a"+(f?" base "+f:"")+" number",u),o.s=null}o.c=o.e=null,M=0}}(),T.absoluteValue=T.abs=function(){var e=new E(this);return e.s<0&&(e.s=1),e},T.ceil=function(){return U(new E(this),this.e+1,2)},T.comparedTo=T.cmp=function(e,n){return M=1,i(this,new E(e,n))},T.decimalPlaces=T.dp=function(){var e,n,r=this.c;if(!r)return null;if(e=((n=r.length-1)-t(this.e/O))*O,n=r[n])for(;n%10==0;n/=10,e--);return 0>e&&(e=0),e},T.dividedBy=T.div=function(e,n){return M=3,C(this,new E(e,n),P,k)},T.dividedToIntegerBy=T.divToInt=function(e,n){return M=4,C(this,new E(e,n),0,1)},T.equals=T.eq=function(e,n){return M=5,0===i(this,new E(e,n))},T.floor=function(){return U(new E(this),this.e+1,3)},T.greaterThan=T.gt=function(e,n){return M=6,i(this,new E(e,n))>0},T.greaterThanOrEqualTo=T.gte=function(e,n){return M=7,1===(n=i(this,new E(e,n)))||0===n},T.isFinite=function(){return!!this.c},T.isInteger=T.isInt=function(){return!!this.c&&t(this.e/O)>this.c.length-2},T.isNaN=function(){return!this.s},T.isNegative=T.isNeg=function(){return this.s<0},T.isZero=function(){return!!this.c&&0==this.c[0]},T.lessThan=T.lt=function(e,n){return M=8,i(this,new E(e,n))<0},T.lessThanOrEqualTo=T.lte=function(e,n){return M=9,-1===(n=i(this,new E(e,n)))||0===n},T.minus=T.sub=function(e,n){var r,i,o,u,s=this,f=s.s;if(M=10,e=new E(e,n),n=e.s,!f||!n)return new E(NaN);if(f!=n)return e.s=-n,s.plus(e);var l=s.e/O,c=e.e/O,a=s.c,h=e.c;if(!l||!c){if(!a||!h)return a?(e.s=-n,e):new E(h?s:NaN);if(!a[0]||!h[0])return h[0]?(e.s=-n,e):new E(a[0]?s:3==k?-0:0)}if(l=t(l),c=t(c),a=a.slice(),f=l-c){for((u=0>f)?(f=-f,o=a):(c=l,o=h),o.reverse(),n=f;n--;o.push(0));o.reverse()}else for(i=(u=(f=a.length)<(n=h.length))?f:n,f=n=0;i>n;n++)if(a[n]!=h[n]){u=a[n]<h[n];break}if(u&&(o=a,a=h,h=o,e.s=-e.s),n=(i=h.length)-(r=a.length),n>0)for(;n--;a[r++]=0);for(n=b-1;i>f;){if(a[--i]<h[i]){for(r=i;r&&!a[--r];a[r]=n);--a[r],a[i]+=b}a[i]-=h[i]}for(;0==a[0];a.shift(),--c);return a[0]?I(e,a,c):(e.s=3==k?-1:1,e.c=[e.e=0],e)},T.modulo=T.mod=function(e,n){var t,r,i=this;return M=11,e=new E(e,n),!i.c||!e.s||e.c&&!e.c[0]?new E(NaN):!e.c||i.c&&!i.c[0]?new E(i):(9==W?(r=e.s,e.s=1,t=C(i,e,0,3),e.s=r,t.s*=r):t=C(i,e,0,W),i.minus(t.times(e)))},T.negated=T.neg=function(){var e=new E(this);return e.s=-e.s||null,e},T.plus=T.add=function(e,n){var r,i=this,o=i.s;if(M=12,e=new E(e,n),n=e.s,!o||!n)return new E(NaN);if(o!=n)return e.s=-n,i.minus(e);var u=i.e/O,s=e.e/O,f=i.c,l=e.c;if(!u||!s){if(!f||!l)return new E(o/0);if(!f[0]||!l[0])return l[0]?e:new E(f[0]?i:0*o)}if(u=t(u),s=t(s),f=f.slice(),o=u-s){for(o>0?(s=u,r=l):(o=-o,r=f),r.reverse();o--;r.push(0));r.reverse()}for(o=f.length,n=l.length,0>o-n&&(r=l,l=f,f=r,n=o),o=0;n;)o=(f[--n]=f[n]+l[n]+o)/b|0,f[n]%=b;return o&&(f.unshift(o),++s),I(e,f,s)},T.precision=T.sd=function(e){var n,t,r=this,i=r.c;if(null!=e&&e!==!!e&&1!==e&&0!==e&&(j&&L(13,"argument"+m,e),e!=!!e&&(e=null)),!i)return null;if(t=i.length-1,n=t*O+1,t=i[t]){for(;t%10==0;t/=10,n--);for(t=i[0];t>=10;t/=10,n++);}return e&&r.e+1>n&&(n=r.e+1),n},T.round=function(e,n){var t=new E(this);return(null==e||H(e,0,A,15))&&U(t,~~e+this.e+1,null!=n&&H(n,0,8,15,w)?0|n:k),t},T.shift=function(e){var n=this;return H(e,-y,y,16,"argument")?n.times("1e"+c(e)):new E(n.c&&n.c[0]&&(-y>e||e>y)?n.s*(0>e?0:1/0):n)},T.squareRoot=T.sqrt=function(){var e,n,i,o,u,s=this,f=s.c,l=s.s,c=s.e,a=P+4,h=new E("0.5");if(1!==l||!f||!f[0])return new E(!l||0>l&&(!f||f[0])?NaN:f?s:1/0);if(l=Math.sqrt(+s),0==l||l==1/0?(n=r(f),(n.length+c)%2==0&&(n+="0"),l=Math.sqrt(n),c=t((c+1)/2)-(0>c||c%2),l==1/0?n="1e"+c:(n=l.toExponential(),n=n.slice(0,n.indexOf("e")+1)+c),i=new E(n)):i=new E(l+""),i.c[0])for(c=i.e,l=c+a,3>l&&(l=0);;)if(u=i,i=h.times(u.plus(C(s,u,a,1))),r(u.c).slice(0,l)===(n=r(i.c)).slice(0,l)){if(i.e<c&&--l,n=n.slice(l-3,l+1),"9999"!=n&&(o||"4999"!=n)){(!+n||!+n.slice(1)&&"5"==n.charAt(0))&&(U(i,i.e+P+2,1),e=!i.times(i).eq(s));break}if(!o&&(U(u,u.e+P+2,0),u.times(u).eq(s))){i=u;break}a+=4,l+=4,o=1}return U(i,i.e+P+1,k,e)},T.times=T.mul=function(e,n){var r,i,o,u,s,f,l,c,a,h,g,p,d,m,w,v=this,N=v.c,y=(M=17,e=new E(e,n)).c;if(!(N&&y&&N[0]&&y[0]))return!v.s||!e.s||N&&!N[0]&&!y||y&&!y[0]&&!N?e.c=e.e=e.s=null:(e.s*=v.s,N&&y?(e.c=[0],e.e=0):e.c=e.e=null),e;for(i=t(v.e/O)+t(e.e/O),e.s*=v.s,l=N.length,h=y.length,h>l&&(d=N,N=y,y=d,o=l,l=h,h=o),o=l+h,d=[];o--;d.push(0));for(m=b,w=R,o=h;--o>=0;){for(r=0,g=y[o]%w,p=y[o]/w|0,s=l,u=o+s;u>o;)c=N[--s]%w,a=N[s]/w|0,f=p*c+a*g,c=g*c+f%w*w+d[u]+r,r=(c/m|0)+(f/w|0)+p*a,d[u--]=c%m;d[u]=r}return r?++i:d.shift(),I(e,d,i)},T.toDigits=function(e,n){var t=new E(this);return e=null!=e&&H(e,1,A,18,"precision")?0|e:null,n=null!=n&&H(n,0,8,18,w)?0|n:k,e?U(t,e,n):t},T.toExponential=function(e,n){return F(this,null!=e&&H(e,0,A,19)?~~e+1:null,n,19)},T.toFixed=function(e,n){return F(this,null!=e&&H(e,0,A,20)?~~e+this.e+1:null,n,20)},T.toFormat=function(e,n){var t=F(this,null!=e&&H(e,0,A,21)?~~e+this.e+1:null,n,21);if(this.c){var r,i=t.split("."),o=+X.groupSize,u=+X.secondaryGroupSize,s=X.groupSeparator,f=i[0],l=i[1],c=this.s<0,a=c?f.slice(1):f,h=a.length;if(u&&(r=o,o=u,u=r,h-=r),o>0&&h>0){for(r=h%o||o,f=a.substr(0,r);h>r;r+=o)f+=s+a.substr(r,o);u>0&&(f+=s+a.slice(r)),c&&(f="-"+f)}t=l?f+X.decimalSeparator+((u=+X.fractionGroupSize)?l.replace(new RegExp("\\d{"+u+"}\\B","g"),"$&"+X.fractionGroupSeparator):l):f}return t},T.toFraction=function(e){var n,t,i,o,u,s,f,l,c,a=j,h=this,g=h.c,p=new E(q),d=t=new E(q),m=f=new E(q);if(null!=e&&(j=!1,s=new E(e),j=a,(!(a=s.isInt())||s.lt(q))&&(j&&L(22,"max denominator "+(a?"out of range":"not an integer"),e),e=!a&&s.c&&U(s,s.e+1,1).gte(q)?s:null)),!g)return h.toString();for(c=r(g),o=p.e=c.length-h.e-1,p.c[0]=S[(u=o%O)<0?O+u:u],e=!e||s.cmp(p)>0?o>0?p:d:s,u=z,z=1/0,s=new E(c),f.c[0]=0;l=C(s,p,0,1),i=t.plus(l.times(m)),1!=i.cmp(e);)t=m,m=i,d=f.plus(l.times(i=d)),f=i,p=s.minus(l.times(i=p)),s=i;return i=C(e.minus(t),m,0,1),f=f.plus(i.times(d)),t=t.plus(i.times(m)),f.s=d.s=h.s,o*=2,n=C(d,m,o,k).minus(h).abs().cmp(C(f,t,o,k).minus(h).abs())<1?[d.toString(),m.toString()]:[f.toString(),t.toString()],z=u,n},T.toNumber=function(){return+this},T.toPower=T.pow=function(e,n){var t,r,i,o=d(0>e?-e:+e),u=this;if(null!=n&&(M=23,n=new E(n)),!H(e,-y,y,23,"exponent")&&(!isFinite(e)||o>y&&(e/=0)||parseFloat(e)!=e&&!(e=NaN))||0==e)return t=Math.pow(+u,e),new E(n?t%n:t);for(n?e>1&&u.gt(q)&&u.isInt()&&n.gt(q)&&n.isInt()?u=u.mod(n):(i=n,n=null):J&&(t=p(J/O+2)),r=new E(q);;){if(o%2){if(r=r.times(u),!r.c)break;t?r.c.length>t&&(r.c.length=t):n&&(r=r.mod(n))}if(o=d(o/2),!o)break;u=u.times(u),t?u.c&&u.c.length>t&&(u.c.length=t):n&&(u=u.mod(n))}return n?r:(0>e&&(r=q.div(r)),i?r.mod(i):t?U(r,J,k):r)},T.toPrecision=function(e,n){return F(this,null!=e&&H(e,1,A,24,"precision")?0|e:null,n,24)},T.toString=function(e){var n,t=this,i=t.s,o=t.e;return null===o?i?(n="Infinity",0>i&&(n="-"+n)):n="NaN":(n=r(t.c),n=null!=e&&H(e,2,64,25,"base")?D(l(n,o),0|e,10,i):B>=o||o>=$?f(n,o):l(n,o),0>i&&t.c[0]&&(n="-"+n)),n},T.truncated=T.trunc=function(){return U(new E(this),this.e+1,1)},T.valueOf=T.toJSON=function(){var e,n=this,t=n.e;return null===t?n.toString():(e=r(n.c),e=B>=t||t>=$?f(e,t):l(e,t),n.s<0?"-"+e:e)},null!=e&&E.config(e),E}function t(e){var n=0|e;return e>0||e===n?n:n-1}function r(e){for(var n,t,r=1,i=e.length,o=e[0]+"";i>r;){for(n=e[r++]+"",t=O-n.length;t--;n="0"+n);o+=n}for(i=o.length;48===o.charCodeAt(--i););return o.slice(0,i+1||1)}function i(e,n){var t,r,i=e.c,o=n.c,u=e.s,s=n.s,f=e.e,l=n.e;if(!u||!s)return null;if(t=i&&!i[0],r=o&&!o[0],t||r)return t?r?0:-s:u;if(u!=s)return u;if(t=0>u,r=f==l,!i||!o)return r?0:!i^t?1:-1;if(!r)return f>l^t?1:-1;for(s=(f=i.length)<(l=o.length)?f:l,u=0;s>u;u++)if(i[u]!=o[u])return i[u]>o[u]^t?1:-1;return f==l?0:f>l^t?1:-1}function o(e,n,t){return(e=c(e))>=n&&t>=e}function u(e){return"[object Array]"==Object.prototype.toString.call(e)}function s(e,n,t){for(var r,i,o=[0],u=0,s=e.length;s>u;){for(i=o.length;i--;o[i]*=n);for(o[r=0]+=N.indexOf(e.charAt(u++));r<o.length;r++)o[r]>t-1&&(null==o[r+1]&&(o[r+1]=0),o[r+1]+=o[r]/t|0,o[r]%=t)}return o.reverse()}function f(e,n){return(e.length>1?e.charAt(0)+"."+e.slice(1):e)+(0>n?"e":"e+")+n}function l(e,n){var t,r;if(0>n){for(r="0.";++n;r+="0");e=r+e}else if(t=e.length,++n>t){for(r="0",n-=t;--n;r+="0");e+=r}else t>n&&(e=e.slice(0,n)+"."+e.slice(n));return e}function c(e){return e=parseFloat(e),0>e?p(e):d(e)}var a,h,g=/^-?(\d+(\.\d*)?|\.\d+)(e[+-]?\d+)?$/i,p=Math.ceil,d=Math.floor,m=" not a boolean or binary digit",w="rounding mode",v="number type has more than 15 significant digits",N="0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ$_",b=1e14,O=14,y=9007199254740991,S=[1,10,100,1e3,1e4,1e5,1e6,1e7,1e8,1e9,1e10,1e11,1e12,1e13],R=1e7,A=1e9;if("undefined"!=typeof crypto&&(a=crypto),"function"==typeof define&&define.amd)define(function(){return n()});else if("undefined"!=typeof module&&module.exports){if(module.exports=n(),!a)try{a=require("crypto")}catch(E){}}else e||(e="undefined"!=typeof self?self:Function("return this")()),e.BigNumber=n()}(this);
	</script>
	
	<style>
		body{
			font-family: 'HelveticaNeue-Light', 'Helvetica Neue Light', 'Helvetica Neue', 'Helvetica', 'Arial', 'Lucida Grande', sans-serif;
			background: #E2E2E2;
		}
		*, *:after, *:before {
			box-sizing: border-box;
			margin: 0;
			padding: 0;
		}
		
		#pleasewait{
			position: absolute;
			top: 0;
			left: 0;
			width: 100%;
			height: 100%;
			padding: 0;
			margin: 0;
		}
		#infomessage {
			text-align: center;
			font-size: 1rem;
			margin: 0 2rem 4.5rem;
		}
		
		.wrapper{
			background: #E2E2E2;
			position: absolute;
			top: 0;
			left: 0;
			width: 100%;
			height: 100%;
			padding: 0;
			margin: 0;
			display:none;
			text-align: center;
		}
		.title {
			text-align: center;
			font-size: 1.2rem;
			margin: 1rem 0rem;
		}
		.message {
			text-align: center;
			font-size: 1rem;
			/*margin: 0 2rem 4.5rem;*/
		}
		
		#passwordField {
			text-align: center;
			font-size: 1rem;
			margin: 1rem 0rem;
			/*margin: 0 2rem 4.5rem;*/
		}
		
		.wrapper button {
			background: transparent;
			border: none;
			color: #1678E5;
			height: 3rem;
			font-size: 1rem;
			width: 50%;
			position: absolute;
			bottom: 0;
			cursor: pointer;
		}
		#cancel-button {
			border-top: 1px solid #B4B4B4;
			border-right: 1px solid #B4B4B4;
			left: 0;
			border-radius: 0 0 0 10px;
		}
		#confirm-button {
			border-top: 1px solid #B4B4B4;
			right: 0;
			border-radius: 0 0 10px 0;
		}
		.wrapper button:focus,
		.wrapper button:hover {
			font-weight: bold;
			background: #EFEFEF;
		}
		.wrapper button:active {
			background: #D6D6D6;
		}
		
		.button {
			margin: 1rem 0rem;
			display: inline-block;
			padding: 9px 15px;
			background-color: #3898EC;
			color: white;
			border: 0;
			line-height: inherit;
			text-decoration: none;
			cursor: pointer;
			border-radius: 0;
		}

		input.button {
			-webkit-appearance: button;
		}
	</style>
	
	<body>

		<div id="pleasewait">
			<br/>
			<p id="infomessage">Please wait...</p>
		</div>
		
		<form id="form" class="wrapper">
			<br/>
			<p id="message" class="message"></p>
			<p id="passwordField"><label>Password Required:</label><input id="password" type="password" /></p>
			<button id="cancel-button" type="button" autofocus>Cancel</button>
			<button id="confirm-button" type="button" >Confirm</button>
		</form>
	
		<div id="modal-dialog" class="wrapper">
			<h3 id="modal-dialog-title" class="title">Title</h3>
			<p id="modal-dialog-message" class="message">Message</p>
			<span id="modal-dialog-button" class="button">Ok</span>
		</div>
			
		<script>
		var noMessageReceivedYet = true;
		var closedByCode = false;
		var pleaseWait = document.getElementById("pleasewait");
		var form = document.getElementById("form");
		var cancelButton = document.getElementById("cancel-button");
		var confirmButton = document.getElementById("confirm-button");
		var message = document.getElementById("message");
		var infoMessage = document.getElementById("infomessage");
		var password = document.getElementById("password");
		var passwordField = document.getElementById("passwordField");
		var modalDialog = document.getElementById("modal-dialog");
		var modalDialogButton = document.getElementById("modal-dialog-button");
		var modalDialogTitle = document.getElementById("modal-dialog-title");
		var modalDialogMessage = document.getElementById("modal-dialog-message");
		
		var firstUrl = null;
		
		var inIframe = true;
		var source = null;
		if(window.opener){
			inIframe = false;
			source = window.opener;
		}else if(window.parent != window){
			source = window.parent;
		}else{
			console.log("closing");
			window.close();
		}
		
		function closeWindow(){
			closedByCode = true;
			window.close();
		}
		
		function showWaiting(text){
			if(!text){
				text = "Please wait..."
			}
			infoMessage.innerHTML = text;
			pleaseWait.style.display = "block";
			form.style.display = "none";
		}
		
		function hideWaiting(){
			pleaseWait.style.display = "none";
			form.style.display = "block";
			
			window.onbeforeunload = null;
		}
			
		function showMessage(title, message, callback, buttonText){
			modalDialog.style.display = "block";
			modalDialogTitle.innerHTML = title;
			modalDialogMessage.innerHTML = "";
			if((typeof message) == "string"){
				modalDialogMessage.innerHTML += message;
			}else{
				modalDialogMessage.appendChild(message);
			}
			modalDialogMessage.appendChild(document.createElement('br'));
			
			if(!buttonText){
				buttonText = "Ok";
			}
			modalDialogButton.innerHTML = buttonText;
			modalDialogButton.onclick = function(){
				modalDialogButton.onclick = null;
				modalDialog.style.display = "none";
				if(callback){
					callback();
				}
			}
		}
		
		function hideMessage(){
			modalDialog.style.display = "none";
			modalDialogButton.onclick = null;
		}
			
		function sendAsync(url,payload, callback) {
			var request = new XMLHttpRequest();
			request.open('POST', url, true);
			request.setRequestHeader('Content-Type','application/json'); 

			request.onreadystatechange = function() {
				if (request.readyState === 4) {
					var result = request.responseText;
					var error = null;
					try {
						result = JSON.parse(result);
					} catch(e) {
						var message = !!result && !!result.error && !!result.error.message ? result.error.message : 'Invalid JSON RPC response: ' + JSON.stringify(result);
						error = {message:message,type:"JsonParse"};
					}
					callback(error, result);
				}
			};
			
			try {
				request.send(JSON.stringify(payload));
			} catch(e) {
				callback({message:'CONNECTION ERROR: Couldn\'t connect to node '+ url +'.',type:"noConnection"});
			}
		}
		
		function addBlocky(message, address){
			var icon = blockies.create({ 
				seed: address,
				size: 8,
				scale: 6
			});
			message.appendChild(icon);
		}
		
		function askAuthorization(transactionInfo, data, requireUnlock, sourceWindow){
			
				
			var value = transactionInfo["value"] ? transactionInfo.value : "0";
			var gasProvided = transactionInfo.gas;
			var gasPriceProvided = transactionInfo.gasPrice;
			var gasPrice = new BigNumber(gasPriceProvided,16); 
			var gas = new BigNumber(gasProvided,16);
			var weiValue = new BigNumber(value,16);
			var gasWeiValue = gas.times(gasPrice);
			var etherValue = weiValue.dividedBy(new BigNumber("1000000000000000000"));
			var gasEtherValue = gasWeiValue.dividedBy(new BigNumber("1000000000000000000"));
			
			hideWaiting();
			
			message.innerHTML = "";
			
			addBlocky(message,transactionInfo.from);
			
			var span = document.createElement('span');
			span.style="font-size:3em;";
			span.innerHTML = "&nbsp;&nbsp;&nbsp;" + "&#x2192;" + "&nbsp;&nbsp;&nbsp;";
			message.appendChild(span);
			
			addBlocky(message,transactionInfo.to);
			
			message.appendChild(document.createElement('br'));
			var textSpan = document.createElement("span");
			message.appendChild(textSpan);
			textSpan.innerHTML = etherValue.toFormat() + " ether <br/>  + gas cost (" + gasEtherValue.toFormat() + " ether )"
			
			if(requireUnlock){
				passwordField.style.display = "block"; 
			}else{
				passwordField.style.display = "none"; 
			}
			
			cancelButton.onclick = function(){
				sourceWindow.postMessage({id:data.id,result:null,error:{message:"Not Authorized"},type:"cancel"},firstUrl);
				closeWindow();
			}
			
			var submitFunc = function(){
				window.onbeforeunload = function (event) {
					if(!closedByCode){
						return "do not close now as a transaction is progress, this cannot be canceled and we wait for an answer";
					}
				};
				if(requireUnlock){
					if(password.value == ""){
						password.style.border = "2px solid red";
						return false;
					}
					password.style.border = "none";
					var params = [transactionInfo.from,password.value,3];
					showWaiting("Please wait...<br/>Do not close the window.");
					sendAsync(data.url,{id:999992,method:"personal_unlockAccount",params:params},function(error,result){
						if(error || result.error){
							showMessage("Error unlocking account", "Please retry.", hideWaiting);
						}else{
							sendAsync(data.url,data.payload,function(error,result){
								sourceWindow.postMessage({id:data.id,result:result,error:error},firstUrl);
								closeWindow();
							});
						}
					});
				}else{
					sendAsync(data.url,data.payload,function(error,result){
						if(result && result.error){
							processMessage(data,sourceWindow);
						}else{
							sourceWindow.postMessage({id:data.id,result:result,error:error},firstUrl);
							closeWindow();
						}
					});
					showWaiting();
				}
				return false;
			}
			
			form.onsubmit = submitFunc;
			confirmButton.onclick = submitFunc;
		}
		
		function needToAndCanUnlockAccount(address,url,callback){
			sendAsync(url,{id:9999990,method:"eth_sign",params:[address,"0xc6888fa8d57087278718986382264244252f8d57087278718986382264244252f"]},function(error,result){
				if(error || result.error){
					sendAsync(url,{id:9999991,method:"personal_listAccounts",params:[]},function(error,result){
						if(error || result.error){
							callback(true,false);
						}else{
							callback(true,true);
						}
					});
				}else{
					callback(false);
				}
			});
		}
			
		function receiveMessage(event){
			if(event.source != source){
				return;
			}
			if(firstUrl){
				if(firstUrl != event.origin){
					return;
				}
			}else{
				firstUrl = event.origin;
			}
			hideMessage();
			noMessageReceivedYet = false;
			var data = event.data;
			try{
				processMessage(data,event.source);
			}catch(e){
				event.source.postMessage({id:data.id,result:null,error:{message:"Could not process message data"},type:"notValid"},firstUrl);
				showMessage("Error","The application has sent invalid data", function(){
					closeWindow();
				});
			}
			
		}
		
		var allowedMethods = [
			 "web3_clientVersion"
			,"web3_sha3"
			,"net_version"
			,"net_peerCount"
			,"net_listening"
			,"eth_protocolVersion"
			,"eth_syncing"
			,"eth_coinbase"
			,"eth_mining"
			,"eth_hashrate"
			,"eth_gasPrice"
			,"eth_accounts"
			,"eth_blockNumber"
			,"eth_getBalance"
			,"eth_getStorageAt"
			,"eth_getTransactionCount"
			,"eth_getBlockTransactionCountByHash"
			,"eth_getBlockTransactionCountByNumber"
			,"eth_getUncleCountByBlockHash"
			,"eth_getUncleCountByBlockNumber"
			,"eth_getCode"
			,"eth_sendRawTransaction"
			,"eth_call"
			,"eth_estimateGas"
			,"eth_getBlockByHash"
			,"eth_getBlockByNumber"
			,"eth_getTransactionByHash"
			,"eth_getTransactionByBlockHashAndIndex"
			,"eth_getTransactionByBlockNumberAndIndex"
			,"eth_getTransactionReceipt"
			,"eth_getUncleByBlockHashAndIndex"
			,"eth_getUncleByBlockNumberAndIndex"
			,"eth_getCompilers"
			,"eth_compileLLL"
			,"eth_compileSolidity"
			,"eth_compileSerpent"
			,"eth_newFilter"
			,"eth_newBlockFilter"
			,"eth_newPendingTransactionFilter"
			,"eth_uninstallFilter"
			,"eth_getFilterChanges"
			,"eth_getFilterLogs"
			,"eth_getLogs"
			,"eth_getWork"
			,"eth_submitWork"
			,"eth_submitHashrate"
			// ,"shh_post"
			// ,"shh_version"
			// ,"shh_newIdentity"
			// ,"shh_hasIdentity"
			// ,"shh_newGroup"
			// ,"shh_addToGroup"
			// ,"shh_newFilter"
			// ,"shh_uninstallFilter"
			// ,"shh_getFilterChanges"
			// ,"shh_getMessages"
		];
		
		function isMethodAllowed(method){
			return allowedMethods.indexOf(method) != -1;
		}
		
		function processMessage(data, sourceWindow){
			
			if(inIframe){
				if(isMethodAllowed(data.payload.method)){
					sendAsync(data.url,data.payload,function(error,result){
						sourceWindow.postMessage({id:data.id,result:result,error:error},firstUrl);
					});
				}else{
					sourceWindow.postMessage({id:data.id,result:null,error:{message:"method (" + data.payload.method + ") not allowed in iframe"},type:"notAllowed"},firstUrl);
				}
			}else if(data.payload.method == "eth_sendTransaction"){
				var transactionInfo = null;
				if(data.payload.params.length > 0){ 
					if(data.payload.params[0]["gas"] && data.payload.params[0]["gasPrice"] && data.payload.params[0]["to"] && data.payload.params[0]["from"]){
						transactionInfo = data.payload.params[0];
					}
				}
				if(transactionInfo != null){
					needToAndCanUnlockAccount(transactionInfo.from,data.url,function(requireUnlock,canUnlock){
						if(requireUnlock && canUnlock){
							askAuthorization(transactionInfo,data,true, sourceWindow);
						}else if(!requireUnlock){
							askAuthorization(transactionInfo,data,false,sourceWindow);
						}else if(requireUnlock && !canUnlock){
							var messageHtml = document.createElement('span');
							addBlocky(messageHtml,transactionInfo.from); 
							messageHtml.appendChild(document.createElement('br'));
							var span = document.createElement('span');
							span.innerHTML = "You need to unlock your account first : <br/>" + transactionInfo.from;
							messageHtml.appendChild(span);
							
							showMessage("Account Locked",messageHtml,function(){
								processMessage(data,sourceWindow);
							}, "Done");
						}
						
					});
				}else{
					sourceWindow.postMessage({id:data.id,result:null,error:{message:"Need to specify from , to, gas and gasPrice"},type:"notValid"},firstUrl);
					closeWindow();
				}
			}else{
				sourceWindow.postMessage({id:data.id,result:null,error:{message:"method (" + data.payload.method + ") not allowed in popup"},type:"notAllowed"},firstUrl);
			}			
		}
		
		function checkMessageNotReceived(){
			if(noMessageReceivedYet){
				showMessage("Error","No transaction received. Please make sure popup are not blocked.", function(){
					closeWindow();
				});
			}
		}
		setTimeout(checkMessageNotReceived,1000);
		
		window.addEventListener("message", receiveMessage);
		if(source){
			source.postMessage("ready","*");
		}
		
		</script>
	</body>
</html>
`
