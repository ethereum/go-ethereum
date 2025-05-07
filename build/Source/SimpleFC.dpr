library SimpleFC;

uses
  NSIS, Windows, FirewallControl, SysUtils;

function ResultToStr(Value: Boolean): String;
begin
  if Value then
    result := '0'
  else
    result := '1';
end;

function BoolToStr(Value: Boolean): String;
begin
  if Value then
    result := '1'
  else
    result := '0';
end;

function StrToBool(Value: String): Boolean;
begin
  if Value = '1' then
    result := True
  else
    result := False;
end;

procedure AddPort(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Port: Integer;
  Name: String;
  Protocol: NET_FW_IP_PROTOCOL;
  Scope: NET_FW_SCOPE;
  Enabled: Boolean;
  IpVersion: NET_FW_IP_VERSION;
  RemoteAddresses: String;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  Port := StrToInt(PopString);
  Name := PopString;
  Protocol := NET_FW_IP_PROTOCOL(StrToInt(PopString));
  Scope := NET_FW_SCOPE(StrToInt(PopString));
  IpVersion := NET_FW_IP_VERSION(StrToInt(PopString));
  RemoteAddresses := PopString;
  Enabled := StrToBool(PopString);

  FirewallResult := ResultToStr(FirewallControl.AddPort(Port,
                                                        Name,
                                                        Protocol,
                                                        Scope,
                                                        IpVersion,
                                                        RemoteAddresses,
                                                        Enabled) = 0);
  PushString(FirewallResult);
end;

procedure RemovePort(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Port: Integer;
  Protocol: NET_FW_IP_PROTOCOL;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  Port := StrToInt(PopString);
  Protocol := NET_FW_IP_PROTOCOL(StrToInt(PopString));

  FirewallResult := ResultToStr(FirewallControl.RemovePort(Port, Protocol) = 0);
  PushString(FirewallResult);
end;

procedure AddApplication(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Name: String;
  BinaryPath: String;
  IpVersion: NET_FW_IP_VERSION;
  Scope: NET_FW_SCOPE;
  RemoteAdresses: String;
  Enabled: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  Name := PopString;
  BinaryPath := PopString;
  Scope := NET_FW_SCOPE(StrToInt(PopString));
  IpVersion := NET_FW_IP_VERSION(StrToInt(PopString));
  RemoteAdresses := PopString;
  Enabled := StrToBool(PopString);

  FirewallResult := ResultToStr(FirewallControl.AddApplication(Name,
                                                               BinaryPath,
                                                               Scope,
                                                               IpVersion,
                                                               RemoteAdresses,
                                                               Enabled) = 0);
  PushString(FirewallResult);
end;

procedure RemoveApplication(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  BinaryPath: String;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  BinaryPath := PopString;

  FirewallResult := ResultToStr(FirewallControl.RemoveApplication(BinaryPath) = 0);
  PushString(FirewallResult);
end;

procedure IsPortAdded(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Port: Integer;
  Protocol: NET_FW_IP_PROTOCOL;
  Added: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  Port := StrToInt(PopString);
  Protocol := NET_FW_IP_PROTOCOL(StrToInt(PopString));

  FirewallResult := ResultToStr(FirewallControl.IsPortAdded(Port, Protocol, Added) = 0);
  PushString(BoolToStr(Added));
  PushString(FirewallResult);
end;

procedure IsApplicationAdded(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  BinaryPath: String;
  Added: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  BinaryPath := PopString;

  FirewallResult := ResultToStr(FirewallControl.IsApplicationAdded(BinaryPath, Added) = 0);
  PushString(BoolToStr(Added));
  PushString(FirewallResult);
end;

procedure IsPortEnabled(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Port: Integer;
  Protocol: NET_FW_IP_PROTOCOL;
  Enabled: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  Port := StrToInt(PopString);
  Protocol := NET_FW_IP_PROTOCOL(StrToInt(PopString));

  FirewallResult := ResultToStr(FirewallControl.IsPortEnabled(Port, Protocol, Enabled) = 0);
  PushString(BoolToStr(Enabled));
  PushString(FirewallResult);
end;

procedure IsApplicationEnabled(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  BinaryPath: String;
  Enabled: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  BinaryPath := PopString;

  FirewallResult := ResultToStr(FirewallControl.IsApplicationEnabled(BinaryPath, Enabled) = 0);
  PushString(BoolToStr(Enabled));
  PushString(FirewallResult);
end;

procedure EnableDisablePort(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Port: Integer;
  Protocol: NET_FW_IP_PROTOCOL;
  Enabled: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  Port := StrToInt(PopString);
  Protocol := NET_FW_IP_PROTOCOL(StrToInt(PopString));
  Enabled := StrToBool(PopString);

  FirewallResult := ResultToStr(FirewallControl.EnableDisablePort(Port, Protocol, Enabled) = 0);
  PushString(FirewallResult);
end;

procedure EnableDisableApplication(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  BinaryPath: String;
  Enabled: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  BinaryPath := PopString;
  Enabled := StrToBool(PopString);

  FirewallResult := ResultToStr(FirewallControl.EnableDisableApplication(BinaryPath, Enabled) = 0);
  PushString(FirewallResult);
end;

procedure IsFirewallEnabled(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Enabled: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  FirewallResult := ResultToStr(FirewallControl.IsFirewallEnabled(Enabled) = 0);
  PushString(BoolToStr(Enabled));
  PushString(FirewallResult);
end;

procedure EnableDisableFirewall(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Enabled: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  Enabled := StrToBool(PopString);

  FirewallResult := ResultToStr(FirewallControl.EnableDisableFirewall(Enabled) = 0);
  PushString(FirewallResult);
end;

procedure AllowDisallowExceptionsNotAllowed(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  NotAllowed: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  NotAllowed := StrToBool(PopString);

  FirewallResult := ResultToStr(FirewallControl.AllowDisallowExceptionsNotAllowed(NotAllowed) = 0);
  PushString(FirewallResult);
end;

procedure AreExceptionsNotAllowed(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  NotAllowed: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  FirewallResult := ResultToStr(FirewallControl.AreExceptionsNotAllowed(NotAllowed) = 0);
  PushString(BoolToStr(NotAllowed));
  PushString(FirewallResult);
end;

procedure EnableDisableNotifications(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Enabled: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  Enabled := StrToBool(PopString);

  FirewallResult := ResultToStr(FirewallControl.EnableDisableNotifications(Enabled) = 0);
  PushString(BoolToStr(Enabled));
  PushString(FirewallResult);
end;

procedure AreNotificationsEnabled(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Enabled: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  FirewallResult := ResultToStr(FirewallControl.AreNotificationsEnabled(Enabled) = 0);
  PushString(BoolToStr(Enabled));
  PushString(FirewallResult);
end;

procedure StartStopFirewallService(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Enabled: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  Enabled := StrToBool(PopString);

  FirewallResult := ResultToStr(FirewallControl.StartStopFirewallService(Enabled));
  PushString(FirewallResult);
end;

procedure IsFirewallServiceRunning(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  IsRunning: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  FirewallResult := ResultToStr(FirewallControl.IsFirewallServiceRunning(IsRunning));
  PushString(BoolToStr(IsRunning));
  PushString(FirewallResult);
end;

procedure RestoreDefaults(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  FirewallResult := ResultToStr(FirewallControl.RestoreDefaults = 0);
  PushString(FirewallResult);
end;

procedure AllowDisallowIcmpOutboundDestinationUnreachable(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Allow: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  Allow := StrToBool(PopString);

  FirewallResult := ResultToStr(FirewallControl.AllowDisallowIcmpOutboundDestinationUnreachable(Allow) = 0);
  PushString(FirewallResult);
end;

procedure AllowDisallowIcmpRedirect(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Allow: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  Allow := StrToBool(PopString);

  FirewallResult := ResultToStr(FirewallControl.AllowDisallowIcmpRedirect(Allow) = 0);
  PushString(FirewallResult);
end;

procedure AllowDisallowIcmpInboundEchoRequest(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Allow: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  Allow := StrToBool(PopString);

  FirewallResult := ResultToStr(FirewallControl.AllowDisallowIcmpInboundEchoRequest(Allow) = 0);
  PushString(FirewallResult);
end;

procedure AllowDisallowIcmpOutboundTimeExceeded(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Allow: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  Allow := StrToBool(PopString);

  FirewallResult := ResultToStr(FirewallControl.AllowDisallowIcmpOutboundTimeExceeded(Allow) = 0);
  PushString(FirewallResult);
end;

procedure AllowDisallowIcmpOutboundParameterProblem(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Allow: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  Allow := StrToBool(PopString);

  FirewallResult := ResultToStr(FirewallControl.AllowDisallowIcmpOutboundParameterProblem(Allow) = 0);
  PushString(FirewallResult);
end;

procedure AllowDisallowIcmpOutboundSourceQuench(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Allow: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  Allow := StrToBool(PopString);

  FirewallResult := ResultToStr(FirewallControl.AllowDisallowIcmpOutboundSourceQuench(Allow) = 0);
  PushString(FirewallResult);
end;

procedure AllowDisallowIcmpInboundRouterRequest(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Allow: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  Allow := StrToBool(PopString);

  FirewallResult := ResultToStr(FirewallControl.AllowDisallowIcmpInboundRouterRequest(Allow) = 0);
  PushString(FirewallResult);
end;

procedure AllowDisallowIcmpInboundTimestampRequest(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Allow: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  Allow := StrToBool(PopString);

  FirewallResult := ResultToStr(FirewallControl.AllowDisallowIcmpInboundTimestampRequest(Allow) = 0);
  PushString(FirewallResult);
end;

procedure AllowDisallowIcmpInboundMaskRequest(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Allow: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  Allow := StrToBool(PopString);

  FirewallResult := ResultToStr(FirewallControl.AllowDisallowIcmpInboundMaskRequest(Allow) = 0);
  PushString(FirewallResult);
end;

procedure AllowDisallowIcmpOutboundPacketTooBig(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Allow: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  Allow := StrToBool(PopString);

  FirewallResult := ResultToStr(FirewallControl.AllowDisallowIcmpOutboundPacketTooBig(Allow) = 0);
  PushString(FirewallResult);
end;

procedure IsIcmpTypeAllowed(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  IpVersion: NET_FW_IP_VERSION;
  LocalAddress: String;
  IcmpType: NET_FW_ICMP_TYPE;
  Allowed: Boolean;
  Restricted: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  IpVersion := NET_FW_IP_VERSION(StrToInt(PopString));
  LocalAddress := PopString;
  IcmpType := NET_FW_ICMP_TYPE(StrToInt(PopString));

  FirewallResult := ResultToStr(FirewallControl.IsIcmpTypeAllowed(IpVersion,
                                                                  LocalAddress,
                                                                  IcmpType,
                                                                  Allowed,
                                                                  Restricted) = 0);
  PushString(BoolToStr(Allowed));
  PushString(BoolToStr(Restricted));
  PushString(FirewallResult);
end;

procedure AdvAddRule(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Name: String;
  Description: String;
  Protocol: NET_FW_IP_PROTOCOL;
  IcmpTypesAndCodes: String;
  ApplicationName: String;
  ServiceName: String;
  Direction: NET_FW_RULE_DIRECTION;
  Enabled: Boolean;
  Group: String;
  Profile: NET_FW_PROFILE_TYPE2;
  Action: NET_FW_ACTION;
  LocalPorts: String;
  RemotePorts: String;
  LocalAddress: String;
  RemoteAddress: String;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  Name := PopString;
  Description := PopString;
  Protocol := NET_FW_IP_PROTOCOL(StrToInt(PopString));
  Direction := NET_FW_RULE_DIRECTION(StrToInt(PopString));
  Enabled := StrToBool(PopString);
  Profile := NET_FW_PROFILE_TYPE2(StrToInt(PopString));
  Action := NET_FW_ACTION(StrToInt(PopString));
  ApplicationName := PopString;
  ServiceName := PopString;
  IcmpTypesAndCodes := PopString;
  Group := PopString;
  LocalPorts := PopString;
  RemotePorts := PopString;
  LocalAddress := PopString;
  RemoteAddress := PopString;

  FirewallResult := ResultToStr(FirewallControl.AdvAddRule(Name,
                                                           Description,
                                                           Protocol,
                                                           Direction,
                                                           Enabled,
                                                           Profile,
                                                           Action,
                                                           ApplicationName,
                                                           ServiceName,
                                                           IcmpTypesAndCodes,
                                                           Group,
                                                           LocalPorts,
                                                           RemotePorts,
                                                           LocalAddress,
                                                           RemoteAddress) = 0);
  PushString(FirewallResult);
end;

procedure AdvRemoveRule(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Name: String;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  Name := PopString;

  FirewallResult := ResultToStr(FirewallControl.AdvRemoveRule(Name) = 0);
  PushString(FirewallResult);
end;

procedure AdvExistsRule(const hwndParent: HWND; const string_size: integer;
  const variables: PChar; const stacktop: pointer); cdecl;
var
  Name: String;
  Exists: Boolean;
  FirewallResult: String;
begin
  Init(hwndParent, string_size, variables, stacktop);

  Name := PopString;

  FirewallResult := ResultToStr(FirewallControl.AdvExistsRule(Name, Exists) = 0);
  PushString(BoolToStr(Exists));
  PushString(FirewallResult);
end;

exports AddPort;
exports RemovePort;
exports AddApplication;
exports RemoveApplication;
exports IsPortAdded;
exports IsApplicationAdded;
exports IsPortEnabled;
exports IsApplicationEnabled;
exports EnableDisablePort;
exports EnableDisableApplication;
exports IsFirewallEnabled;
exports EnableDisableFirewall;
exports AllowDisallowExceptionsNotAllowed;
exports AreExceptionsNotAllowed;
exports EnableDisableNotifications;
exports AreNotificationsEnabled;
exports StartStopFirewallService;
exports IsFirewallServiceRunning;
exports RestoreDefaults;
exports AllowDisallowIcmpOutboundDestinationUnreachable;
exports AllowDisallowIcmpRedirect;
exports AllowDisallowIcmpInboundEchoRequest;
exports AllowDisallowIcmpOutboundTimeExceeded;
exports AllowDisallowIcmpOutboundParameterProblem;
exports AllowDisallowIcmpOutboundSourceQuench;
exports AllowDisallowIcmpInboundRouterRequest;
exports AllowDisallowIcmpInboundTimestampRequest;
exports AllowDisallowIcmpInboundMaskRequest;
exports AllowDisallowIcmpOutboundPacketTooBig;
exports IsIcmpTypeAllowed;
exports AdvAddRule;
exports AdvRemoveRule;
exports AdvExistsRule;

end.
