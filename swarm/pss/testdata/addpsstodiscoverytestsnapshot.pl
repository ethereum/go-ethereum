#!/usr/bin/perl

use JSON;

my $f;
my $jsontext;
my $nodelist;
my $network;

open($f, "<", $ARGV[0]) || die "cant open " . $ARGV[0];
while (<$f>) {
	$jsontext .= $_;
}
close($f);

$network = decode_json($jsontext);
$nodelist = $network->{'nodes'};

for ($i = 0; $i < 0+@$nodelist; $i++) {
	#my $protocollist = $$nodelist[$i]{'node'}{'info'}{'protocols'};
	#$$protocollist{'pss'} = "pss";
	my $svc = $$nodelist[$i]{'node'}{'config'}{'services'};
	pop(@$svc);
	push(@$svc, "pss");
	push(@$svc, "bzz");
}

print encode_json($network);
