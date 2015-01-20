extern char *Sha3(char *, int);
char *sha3_cgo(char *data, int l)
{
	return Sha3(data, l);
}
