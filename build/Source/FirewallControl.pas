{
License Agreement

This content is subject to the Mozilla Public License Version 1.1 (the "License");
You may not use this plugin except in compliance with the License. You may 
obtain a copy of the License at http://www.mozilla.org/MPL. 

Alternatively, you may redistribute this library, use and/or modify it 
under the terms of the GNU Lesser General Public License as published 
by the Free Software Foundation; either version 2.1 of the License, 
or (at your option) any later version. You may obtain a copy 
of the LGPL at www.gnu.org/copyleft. 

Software distributed under the License is distributed on an "AS IS" basis, 
WITHOUT WARRANTY OF ANY KIND, either express or implied. See the License 
for the specific language governing rights and limitations under the License. 

The original code is FirewallControl.pas, released April 16, 2007. 

The initial developer of the original code is Rainer Döpke
(Formerly: Rainer Budde) (http://www.speed-soft.de).

SimpleFC - NSIS Firewall Control Plugin is written, published and maintaned by
Rainer Döpke (rainer@speed-soft.de).
}
unit FirewallControl;

interface

uses
  NetFwTypeLib_TLB, ComObj, ActiveX, Variants, SysUtils, ServiceControl;

type
  NET_FW_IP_VERSION = (
    NET_FW_IP_VERSION_V4 = $00000000,
    NET_FW_IP_VERSION_V6 = $00000001,
    NET_FW_IP_VERSION_ANY = $00000002,
    NET_FW_IP_VERSION_MAX = $00000003
  );

  NET_FW_IP_PROTOCOL = (
    NET_FW_IP_PROTOCOL_ICMP_V4 = $00000001,
    NET_FW_IP_PROTOCOL_ICMP_V6 = $0000003A,
    NET_FW_IP_PROTOCOL_TCP = $00000006,
    NET_FW_IP_PROTOCOL_UDP = $00000011,
    NET_FW_IP_PROTOCOL_ANY = $00000100
  );

  NET_FW_ACTION = (
    NET_FW_ACTION_BLOCK = $00000000,
    NET_FW_ACTION_ALLOW = $00000001,
    NET_FW_ACTION_MAX = $00000002
  );

  NET_FW_SCOPE = (
    NET_FW_SCOPE_ALL = $00000000,
    NET_FW_SCOPE_LOCAL_SUBNET = $00000001,
    NET_FW_SCOPE_CUSTOM = $00000002,
    NET_FW_SCOPE_MAX = $00000003
  );

  NET_FW_PROFILE_TYPE2 = (
    NET_FW_PROFILE2_DOMAIN = $00000001,
    NET_FW_PROFILE2_PRIVATE = $00000002,
    NET_FW_PROFILE2_PUBLIC = $00000004,
    NET_FW_PROFILE2_ALL = $7FFFFFFF
  );

  NET_FW_RULE_DIRECTION = (
    NET_FW_RULE_DIR_IN = $00000001,
    NET_FW_RULE_DIR_OUT = $00000002,
    NET_FW_RULE_DIR_MAX = $00000003
  );

  NET_FW_ICMP_TYPE = (
    NET_FW_ICMP_V4_OUTBOUND_DESTINATION_UNREACHABLE = $00000003,
    NET_FW_ICMP_V4_OUTBOUND_SOURCE_QUENCH = $00000004,
    NET_FW_ICMP_V4_REDIRECT = $00000005,
    NET_FW_ICMP_V4_INBOUND_ECHO_REQUEST = $00000008,
    NET_FW_ICMP_V4_INBOUND_ROUTER_REQUEST = $00000009,
    NET_FW_ICMP_V4_OUTBOUND_TIME_EXCEEDED = $0000000B,
    NET_FW_ICMP_V4_OUTBOUND_PARAMETER_PROBLEM = $0000000C,
    NET_FW_ICMP_V4_INBOUND_TIMESTAMP_REQUEST = $0000000D,
    NET_FW_ICMP_V4_INBOUND_MASK_REQUEST = $00000011,
    NET_FW_ICMP_V6_OUTBOUND_DESTINATION_UNREACHABLE = $00000001,
    NET_FW_ICMP_V6_OUTBOUND_PACKET_TOO_BIG = $00000002,
    NET_FW_ICMP_V6_OUTBOUND_TIME_EXCEEDED = $00000003,
    NET_FW_ICMP_V6_OUTBOUND_PARAMETER_PROBLEM = $00000004,
    NET_FW_ICMP_V6_INBOUND_ECHO_REQUEST = $00000080,
    NET_FW_ICMP_V6_REDIRECT = $00000089
  );
  
  { Functions for Windows Firewall }
  function AddPort(Port: Integer; Name: String; Protocol: NET_FW_IP_PROTOCOL;
    Scope: NET_FW_SCOPE; IpVersion: NET_FW_IP_VERSION; RemoteAddresses: String;
    Enabled: Boolean): HRESULT;
  function RemovePort(Port: Integer; Protocol: NET_FW_IP_PROTOCOL): HRESULT;
  function AddApplication(Name: String; BinaryPath: String; Scope: NET_FW_SCOPE;
    IpVersion: NET_FW_IP_VERSION; RemoteAdresses: String; Enabled: Boolean): HRESULT;
  function RemoveApplication(BinaryPath: String): HRESULT;
  function IsPortAdded(Port: Integer; Protocol: NET_FW_IP_PROTOCOL;
    var Added: Boolean): HRESULT;
  function IsApplicationAdded(BinaryPath: String; var Added: Boolean): HRESULT;
  function IsPortEnabled(Port: Integer; Protocol: NET_FW_IP_PROTOCOL;
    var Enabled: Boolean): HRESULT;
  function IsApplicationEnabled(BinaryPath: String; var Enabled: Boolean): HRESULT;
  function EnableDisablePort(Port: Integer; Protocol: NET_FW_IP_PROTOCOL;
    Enabled: Boolean): HRESULT;
  function EnableDisableApplication(BinaryPath: String; Enabled: Boolean): HRESULT;
  function IsFirewallEnabled(var Enabled: Boolean): HRESULT;
  function EnableDisableFirewall(Enabled: Boolean): HRESULT;
  function AllowDisallowExceptionsNotAllowed(NotAllowed: Boolean): HRESULT;
  function AreExceptionsNotAllowed(var NotAllowed: Boolean): HRESULT;
  function EnableDisableNotifications(Enabled: Boolean): HRESULT;
  function AreNotificationsEnabled(var Enabled: Boolean): HRESULT;
  function IsFirewallServiceRunning(var IsRunning: Boolean): Boolean;
  function StartStopFirewallService(StartService: Boolean): Boolean;
  function RestoreDefaults: HRESULT;
  function AllowDisallowIcmpOutboundDestinationUnreachable(Allow: Boolean): HRESULT;
  function AllowDisallowIcmpRedirect(Allow: Boolean): HRESULT;
  function AllowDisallowIcmpInboundEchoRequest(Allow: Boolean): HRESULT;
  function AllowDisallowIcmpOutboundTimeExceeded(Allow: Boolean): HRESULT;
  function AllowDisallowIcmpOutboundParameterProblem(Allow: Boolean): HRESULT;
  function AllowDisallowIcmpOutboundSourceQuench(Allow: Boolean): HRESULT;
  function AllowDisallowIcmpInboundRouterRequest(Allow: Boolean): HRESULT;
  function AllowDisallowIcmpInboundTimestampRequest(Allow: Boolean): HRESULT;
  function AllowDisallowIcmpInboundMaskRequest(Allow: Boolean): HRESULT;
  function AllowDisallowIcmpOutboundPacketTooBig(Allow: Boolean): HRESULT;
  function IsIcmpTypeAllowed(IpVersion: NET_FW_IP_VERSION; LocalAddress: String;
    IcmpType: NET_FW_ICMP_TYPE; var Allowed: Boolean; var Restricted: Boolean): HRESULT;

  { Functions for Windows Firewall with advanced security }
  function AdvAddRule(Name: String; Description: String;
    Protocol: NET_FW_IP_PROTOCOL; Direction: NET_FW_RULE_DIRECTION;
    Enabled: Boolean; Profile: NET_FW_PROFILE_TYPE2; Action: NET_FW_ACTION;
    ApplicationName: String; ServiceName: String; IcmpTypesAndCodes: String;
    Group: String; LocalPorts: String; RemotePorts: String;
    LocalAddress: String; RemoteAddress: String): HRESULT;
  function AdvRemoveRule(Name: String): HRESULT;
  function AdvExistsRule(Name: String; var Exists: Boolean): HRESULT;

implementation

const
  FW_MGR_CLASS_NAME = 'HNetCfg.FwMgr';
  FW_OPENPORT_CLASS_NAME = 'HNetCfg.FwOpenPort';
  FW_AUTHORIZED_APPLICATION = 'HNetCfg.FwAuthorizedApplication';
  FW_POLICY2_NAME = 'HNetCfg.FwPolicy2';
  FW_RULE_NAME = 'HNetCfg.FWRule';
  FW_SERVICE_XP_WIN2003 = 'SharedAccess';
  FW_SERVICE_VISTA = 'MpsSvc';

function CreateWideString(Value: String): PWideChar;
var
  WideValue: PWideChar;
begin
  GetMem(WideValue, Length(Value) * SizeOf(WideChar) + 1);
  StringToWideChar(Value, WideValue, Length(Value) * SizeOf(WideChar) + 1);

  Result := WideValue;
end;

procedure FreeWideString(Value: PWideChar);
begin
  FreeMem(Value);
end;

function AdvAddRule(Name: String; Description: String;
  Protocol: NET_FW_IP_PROTOCOL; Direction: NET_FW_RULE_DIRECTION;
  Enabled: Boolean; Profile: NET_FW_PROFILE_TYPE2; Action: NET_FW_ACTION;
  ApplicationName: String; ServiceName: String; IcmpTypesAndCodes: String;
  Group: String; LocalPorts: String; RemotePorts: String;
  LocalAddress: String; RemoteAddress: String): HRESULT;
const
  NET_FW_GROUPING = '@firewallapi.dll,-23255';
var
  FwPolicy2Disp: IDispatch;
  FwPolicy2: INetFwPolicy2;
  FwRuleDisp: IDispatch;
  FwRule: INetFwRule;
begin
  Result := S_OK;

  try
    FwPolicy2Disp := CreateOleObject(FW_POLICY2_NAME);
    try
      FwPolicy2 := INetFwPolicy2(FwPolicy2Disp);

      FwRuleDisp := CreateOleObject(FW_RULE_NAME);
      try
        FwRule := INetFwRule(FwRuleDisp);
        FwRule.Name := Name;
        FwRule.Description := Description;
        FwRule.Protocol := Integer(Protocol);
        FwRule.Direction := Integer(Direction);
        FwRule.Enabled := Enabled;
        FwRule.Profiles := Integer(Profile);
        FwRule.Action := TOleEnum(Action);

        if ApplicationName <> '' then
          FwRule.ApplicationName := ApplicationName;

        if ServiceName <> '' then
          FwRule.ServiceName := ServiceName; 

        if IcmpTypesAndCodes <> '' then
          FwRule.IcmpTypesAndCodes := IcmpTypesAndCodes;

        if Group <> '' then
          FwRule.Grouping := Group 
        else
          FwRule.Grouping := NET_FW_GROUPING;
          
        if LocalPorts <> '' then
          FwRule.LocalPorts := LocalPorts;

        if RemotePorts <> '' then
          FwRule.RemotePorts := RemotePorts;

        if LocalAddress <> '' then
          FwRule.LocalAddresses := LocalAddress;

        if RemoteAddress <> '' then
          FwRule.RemoteAddresses := RemoteAddress;
                    
        FwPolicy2.Rules.Add(FwRule);
      finally
        FwRuleDisp := Unassigned;
      end;
    finally
      FwPolicy2Disp := Unassigned;
    end;

  except
    on E:EOleSysError do
     begin
       Result := E.ErrorCode;
     end;
  end;
end;

function AdvRemoveRule(Name: String): HRESULT;
var
  FwPolicy2Disp: IDispatch;
  FwPolicy2: INetFwPolicy2;
begin
  Result := S_OK;

  try
    FwPolicy2Disp := CreateOleObject(FW_POLICY2_NAME);
    try
      FwPolicy2 := INetFwPolicy2(FwPolicy2Disp);
      FwPolicy2.Rules.Remove(Name);
    finally
      FwPolicy2Disp := Unassigned;
    end;

  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function AdvExistsRule(Name: String; var Exists: Boolean): HRESULT;
var
  FwPolicy2Disp: IDispatch;
  FwPolicy2: INetFwPolicy2;
  FwRule: INetFwRule;
  FwRuleInstances: IEnumVariant;
  TempFwRuleObj: OleVariant;
  TempObjValue: Cardinal;
  EnumerateNext: Boolean;
begin
  Result := S_OK;
  EnumerateNext := True;

  try
    FwPolicy2Disp := CreateOleObject(FW_POLICY2_NAME);
    try
      FwPolicy2 := INetFwPolicy2(FwPolicy2Disp);

      FwRuleInstances := FwPolicy2.Rules.Get__NewEnum as IEnumVariant;

      while EnumerateNext and not Exists do
        if FwRuleInstances.Next(1, TempFwRuleObj, TempObjValue) <> 0 then
          EnumerateNext := False
        else
        begin
          FwRule := IUnknown(TempFwRuleObj) as INetFwRule;

          Exists := LowerCase(FwRule.Name) = LowerCase(Name);
        end;
    finally
      FwPolicy2Disp := Unassigned;
    end;

  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function AddPort(Port: Integer; Name: String; Protocol: NET_FW_IP_PROTOCOL;
  Scope: NET_FW_SCOPE; IpVersion: NET_FW_IP_VERSION; RemoteAddresses: String;
  Enabled: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
  FwOpenPortDisp: IDispatch;
  FwOpenPort: INetFwOpenPort;
  RemoteAddressesW: PWideChar;
begin
  Result := S_OK;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
      FwMgr := INetFwMgr(FwMgrDisp);

      FwProfile := FwMgr.LocalPolicy.CurrentProfile;

      FwOpenPortDisp := CreateOleObject(FW_OPENPORT_CLASS_NAME);
      try
        FwOpenPort := INetFwOpenPort(FwOpenPortDisp);

        GetMem(RemoteAddressesW, Length(RemoteAddresses) * SizeOf(WideChar) + 1);
        try
          StringToWideChar(RemoteAddresses, RemoteAddressesW, Length(RemoteAddresses) * SizeOf(WideChar) + 1);

          FwOpenPort.Port := Port;
          FwOpenPort.Name := Name;
          FwOpenPort.Protocol := TOleEnum(Protocol);

          if (Scope = NET_FW_SCOPE_ALL) or (Scope = NET_FW_SCOPE_LOCAL_SUBNET) then
            FwOpenPort.Scope := TOleEnum(Scope)
          else
            FwOpenPort.RemoteAddresses := RemoteAddressesW;

          FwOpenPort.IpVersion := TOleEnum(IpVersion);
          FwOpenPort.Enabled := Enabled;

          FwProfile.GloballyOpenPorts.Add(FwOpenPort);

        finally
          FreeMem(RemoteAddressesW);
        end;

      finally
        FwOpenPortDisp := Unassigned;
      end;
    finally
      FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function RemovePort(Port: Integer; Protocol: NET_FW_IP_PROTOCOL): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
begin
  Result := S_OK;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
      FwMgr := INetFwMgr(FwMgrDisp);

      FwProfile := FwMgr.LocalPolicy.CurrentProfile;
      FwProfile.GloballyOpenPorts.Remove(Port, TOleEnum(Protocol));
    finally
      FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function AddApplication(Name: String; BinaryPath: String; Scope: NET_FW_SCOPE;
  IpVersion: NET_FW_IP_VERSION; RemoteAdresses: String;
  Enabled: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
  FwAppDisp: IDispatch;
  FwApp: INetFwAuthorizedApplication;
  RemoteAddressesW: PWideChar;
begin
  Result := S_OK;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
      FwMgr := INetFwMgr(FwMgrDisp);

      FwProfile := FwMgr.LocalPolicy.CurrentProfile;

      FwAppDisp := CreateOleObject(FW_AUTHORIZED_APPLICATION);
      try
        FwApp := INetFwAuthorizedApplication(FwAppDisp);

        GetMem(RemoteAddressesW, Length(RemoteAdresses) * SizeOf(WideChar) + 1);
        try
          StringToWideChar(RemoteAdresses, RemoteAddressesW, Length(RemoteAdresses) * SizeOf(WideChar) + 1);

          FwApp.Name := Name;
          FwApp.ProcessImageFileName := BinaryPath;

          if (Scope = NET_FW_SCOPE_ALL) or (Scope = NET_FW_SCOPE_LOCAL_SUBNET) then
            FwApp.Scope := TOleEnum(Scope)
          else
            FwApp.RemoteAddresses := RemoteAddressesW;

          FwApp.IpVersion := TOleEnum(IpVersion);
          FwApp.Enabled := Enabled;

          FwProfile.AuthorizedApplications.Add(FwApp);

        finally
          FreeMem(RemoteAddressesW)
        end;
      finally
        FwAppDisp := Unassigned;
      end;
    finally
      FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function RemoveApplication(BinaryPath: String): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
begin
   Result := S_OK;

   try
     FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
     try
       FwMgr := INetFwMgr(FwMgrDisp);

       FwProfile := FwMgr.LocalPolicy.CurrentProfile;
       FwProfile.AuthorizedApplications.Remove(BinaryPath);
     finally
       FwMgrDisp := Unassigned;
     end; 
   except 
     on E:EOleSysError do 
       Result := E.ErrorCode;
   end;
end;

function IsPortAdded(Port: Integer; Protocol: NET_FW_IP_PROTOCOL;
  var Added: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
  FwOpenPort: INetFwOpenPort;
  FwOpenPortInstances: IEnumVariant;
  TempFwPortObj: OleVariant;
  TempObjValue: Cardinal;
  EnumerateNext: Boolean;
begin
  Result := S_OK;
  Added := False;
  EnumerateNext := True;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
      FwMgr := INetFwMgr(FwMgrDisp);

      FwProfile := FwMgr.LocalPolicy.CurrentProfile;
      FwOpenPortInstances := FwProfile.GloballyOpenPorts.Get__NewEnum as IEnumVariant;

      while EnumerateNext and not Added do
        if FwOpenPortInstances.Next(1, TempFwPortObj, TempObjValue) <> 0 then
          EnumerateNext := False
        else
        begin
          FwOpenPort := IUnknown(TempFwPortObj) as INetFwOpenPort;

          Added := (FwOpenPort.Port = Port) and (FwOpenPort.Protocol = TOleEnum(Protocol))
        end;

    finally
      FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function IsApplicationAdded(BinaryPath: String; var Added: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
  FwApp: INetFwAuthorizedApplication;
  FwAppInstances: IEnumVariant;
  TempFwApp: OleVariant;
  TempObjValue: Cardinal;
  EnumerateNext: Boolean;
begin
  Result := S_OK;
  Added := False;
  EnumerateNext := True;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
      FwMgr := INetFwMgr(FwMgrDisp);

      FwProfile := FwMgr.LocalPolicy.CurrentProfile;
      FwAppInstances := FwProfile.AuthorizedApplications.Get__NewEnum as IEnumVariant;

      while EnumerateNext and not Added do
        if FwAppInstances.Next(1, TempFwApp, TempObjValue) <> 0 then
          EnumerateNext := False
        else
        begin
          FwApp := IUnknown(TempFwApp) as INetFwAuthorizedApplication;

          Added := LowerCase(FwApp.ProcessImageFileName) = LowerCase(BinaryPath)
        end;

    finally
      FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function IsPortEnabled(Port: Integer; Protocol: NET_FW_IP_PROTOCOL;
  var Enabled: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
  FwOpenPort: INetFwOpenPort;
  FwOpenPortInstances: IEnumVariant;
  TempFwPortObj: OleVariant;
  TempObjValue: Cardinal;
  EnumerateNext: Boolean;
begin
  Result := S_OK;
  Enabled := False;
  EnumerateNext := True;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
     FwMgr := INetFwMgr(FwMgrDisp);

     FwProfile := FwMgr.LocalPolicy.CurrentProfile;
     FwOpenPortInstances := FwProfile.GloballyOpenPorts.Get__NewEnum as IEnumVariant;

     while EnumerateNext do
       if FwOpenPortInstances.Next(1, TempFwPortObj, TempObjValue) <> 0 then
         EnumerateNext := False
       else
       begin
         FwOpenPort := IUnknown(TempFwPortObj) as INetFwOpenPort;

         if (FwOpenPort.Port = Port) and (FwOpenPort.Protocol = TOleEnum(Protocol)) then
         begin
           Enabled := FwOpenPort.Enabled;
           EnumerateNext := False;
         end;

       end;

    finally
     FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function IsApplicationEnabled(BinaryPath: String;
  var Enabled: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
  FwApp: INetFwAuthorizedApplication;
  FwAppInstances: IEnumVariant;
  TempFwApp: OleVariant;
  TempObjValue: Cardinal;
  EnumerateNext: Boolean;
begin
  Result := S_OK;
  Enabled := False;
  EnumerateNext := True;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
      FwMgr := INetFwMgr(FwMgrDisp);

      FwProfile := FwMgr.LocalPolicy.CurrentProfile;
      FwAppInstances := FwProfile.AuthorizedApplications.Get__NewEnum as IEnumVariant;

      while EnumerateNext do
       if FwAppInstances.Next(1, TempFwApp, TempObjValue) <> 0 then
         EnumerateNext := False
       else
       begin
         FwApp := IUnknown(TempFwApp) as INetFwAuthorizedApplication;

         if LowerCase(FwApp.ProcessImageFileName) = LowerCase(BinaryPath) then
         begin
           Enabled := FwApp.Enabled;
           EnumerateNext := False;
         end;

       end;

    finally
     FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function EnableDisablePort(Port: Integer; Protocol: NET_FW_IP_PROTOCOL;
  Enabled: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
  FwOpenPort: INetFwOpenPort;
  FwOpenPortInstances: IEnumVariant;
  TempFwPortObj: OleVariant;
  TempObjValue: Cardinal;
  EnumerateNext: Boolean;
begin
  Result := S_FALSE;
  EnumerateNext := True;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
     FwMgr := INetFwMgr(FwMgrDisp);

     FwProfile := FwMgr.LocalPolicy.CurrentProfile;
     FwOpenPortInstances := FwProfile.GloballyOpenPorts.Get__NewEnum as IEnumVariant;

     while EnumerateNext do
       if FwOpenPortInstances.Next(1, TempFwPortObj, TempObjValue) <> 0 then
         EnumerateNext := False
       else
       begin
         FwOpenPort := IUnknown(TempFwPortObj) as INetFwOpenPort;

         if (FwOpenPort.Port = Port) and (FwOpenPort.Protocol = TOleEnum(Protocol)) then
         begin
           FwOpenPort.Enabled := Enabled;
           EnumerateNext := False;
           Result := S_OK;
         end;
       end;

    finally
     FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function EnableDisableApplication(BinaryPath: String; Enabled: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
  FwApp: INetFwAuthorizedApplication;
  FwAppInstances: IEnumVariant;
  TempFwApp: OleVariant;
  TempObjValue: Cardinal;
  EnumerateNext: Boolean;
begin
  Result := S_FALSE;
  EnumerateNext := True;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
      FwMgr := INetFwMgr(FwMgrDisp);

      FwProfile := FwMgr.LocalPolicy.CurrentProfile;
      FwAppInstances := FwProfile.AuthorizedApplications.Get__NewEnum as IEnumVariant;
      
      while EnumerateNext do
       if FwAppInstances.Next(1, TempFwApp, TempObjValue) <> 0 then
         EnumerateNext := False
       else
       begin
         FwApp := IUnknown(TempFwApp) as INetFwAuthorizedApplication;

         if LowerCase(FwApp.ProcessImageFileName) = LowerCase(BinaryPath) then
         begin
           FwApp.Enabled := Enabled;
           EnumerateNext := False;
           Result := S_OK;
         end;

       end;

    finally
      FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function IsFirewallEnabled(var Enabled: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
begin
  Result := S_OK;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
     FwMgr := INetFwMgr(FwMgrDisp);

     FwProfile := FwMgr.LocalPolicy.CurrentProfile;
     Enabled := FwProfile.FirewallEnabled
    finally
     FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function EnableDisableFirewall(Enabled: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
begin
  Result := S_OK;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
     FwMgr := INetFwMgr(FwMgrDisp);

     FwProfile := FwMgr.LocalPolicy.CurrentProfile;
     FwProfile.FirewallEnabled := Enabled;
    finally
     FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function AllowDisallowExceptionsNotAllowed(NotAllowed: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
begin
  Result := S_OK;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
     FwMgr := INetFwMgr(FwMgrDisp);

     FwProfile := FwMgr.LocalPolicy.CurrentProfile;
     FwProfile.ExceptionsNotAllowed := NotAllowed;
    finally
     FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function AreExceptionsNotAllowed(var NotAllowed: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
begin
  Result := S_OK;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
     FwMgr := INetFwMgr(FwMgrDisp);

     FwProfile := FwMgr.LocalPolicy.CurrentProfile;
     NotAllowed := FwProfile.ExceptionsNotAllowed;
    finally
     FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function EnableDisableNotifications(Enabled: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
begin
  Result := S_OK;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
     FwMgr := INetFwMgr(FwMgrDisp);

     FwProfile := FwMgr.LocalPolicy.CurrentProfile;
     FwProfile.NotificationsDisabled := not Enabled;
    finally
     FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function AreNotificationsEnabled(var Enabled: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
begin
  Result := S_OK;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
     FwMgr := INetFwMgr(FwMgrDisp);

     FwProfile := FwMgr.LocalPolicy.CurrentProfile;
     Enabled := not FwProfile.NotificationsDisabled;
    finally
     FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function IsFirewallServiceRunning(var IsRunning: Boolean): Boolean;
begin
  IsRunning := False;

  try
    if ServiceControl.ExistsService(FW_SERVICE_VISTA) = 0 then
      if ServiceControl.ServiceIsRunning(FW_SERVICE_VISTA, IsRunning) = 0 then
      begin
        Result := True;
        Exit;
      end;

    if ServiceControl.ExistsService(FW_SERVICE_XP_WIN2003) = 0 then
      if ServiceControl.ServiceIsRunning(FW_SERVICE_XP_WIN2003, IsRunning) = 0 then
      begin
        Result := True;
        Exit;
      end;

    Result := True;
  except
    Result := False;
  end;
end;

function StartStopFirewallService(StartService: Boolean): Boolean;
begin
  Result := False;

  try
    if ServiceControl.ExistsService(FW_SERVICE_VISTA) = 0 then
    begin
      if StartService then
      begin
        if ServiceControl.StartService(FW_SERVICE_VISTA, '') = 0 then
        begin
          Result := True;
          Exit;
        end;
      end
      else
        if ServiceControl.StopService(FW_SERVICE_VISTA) = 0 then
        begin
          Result := True;
          Exit;
        end;
    end;

    if ServiceControl.ExistsService(FW_SERVICE_XP_WIN2003) = 0 then
    begin
      if StartService then
      begin
        if ServiceControl.StartService(FW_SERVICE_XP_WIN2003, '') = 0 then
        begin
          Result := True;
          Exit;
        end;
      end
      else
        if ServiceControl.StopService(FW_SERVICE_XP_WIN2003) = 0 then
        begin
          Result := True;
          Exit;
        end;
    end;

  except
    Result := False;
  end;

end;

function RestoreDefaults: HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
begin
  Result := S_OK;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
      FwMgr := INetFwMgr(FwMgrDisp);
      FwMgr.RestoreDefaults;
    finally
     FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function AllowDisallowIcmpOutboundDestinationUnreachable(Allow: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
begin
  Result := S_OK;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
      FwMgr := INetFwMgr(FwMgrDisp);

      FwProfile := FwMgr.LocalPolicy.CurrentProfile;
      FwProfile.IcmpSettings.AllowOutboundDestinationUnreachable := Allow;
    finally
      FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function AllowDisallowIcmpRedirect(Allow: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
begin
  Result := S_OK;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
      FwMgr := INetFwMgr(FwMgrDisp);

      FwProfile := FwMgr.LocalPolicy.CurrentProfile;
      FwProfile.IcmpSettings.AllowRedirect := Allow;
    finally
      FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function AllowDisallowIcmpInboundEchoRequest(Allow: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
begin
  Result := S_OK;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
      FwMgr := INetFwMgr(FwMgrDisp);

      FwProfile := FwMgr.LocalPolicy.CurrentProfile;
      FwProfile.IcmpSettings.AllowInboundEchoRequest := Allow;
    finally
      FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function AllowDisallowIcmpOutboundTimeExceeded(Allow: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
begin
  Result := S_OK;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
      FwMgr := INetFwMgr(FwMgrDisp);

      FwProfile := FwMgr.LocalPolicy.CurrentProfile;
      FwProfile.IcmpSettings.AllowOutboundTimeExceeded := Allow;
    finally
      FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function AllowDisallowIcmpOutboundParameterProblem(Allow: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
begin
  Result := S_OK;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
      FwMgr := INetFwMgr(FwMgrDisp);

      FwProfile := FwMgr.LocalPolicy.CurrentProfile;
      FwProfile.IcmpSettings.AllowOutboundParameterProblem := Allow;
    finally
      FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function AllowDisallowIcmpOutboundSourceQuench(Allow: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
begin
  Result := S_OK;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
      FwMgr := INetFwMgr(FwMgrDisp);

      FwProfile := FwMgr.LocalPolicy.CurrentProfile;
      FwProfile.IcmpSettings.AllowOutboundSourceQuench := Allow;
    finally
      FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function AllowDisallowIcmpInboundRouterRequest(Allow: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
begin
  Result := S_OK;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
      FwMgr := INetFwMgr(FwMgrDisp);

      FwProfile := FwMgr.LocalPolicy.CurrentProfile;
      FwProfile.IcmpSettings.AllowInboundRouterRequest := Allow;
    finally
      FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function AllowDisallowIcmpInboundTimestampRequest(Allow: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
begin
  Result := S_OK;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
      FwMgr := INetFwMgr(FwMgrDisp);

      FwProfile := FwMgr.LocalPolicy.CurrentProfile;
      FwProfile.IcmpSettings.AllowInboundTimestampRequest := Allow;
    finally
      FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function AllowDisallowIcmpInboundMaskRequest(Allow: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
begin
  Result := S_OK;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
      FwMgr := INetFwMgr(FwMgrDisp);

      FwProfile := FwMgr.LocalPolicy.CurrentProfile;
      FwProfile.IcmpSettings.AllowInboundMaskRequest := Allow;
    finally
      FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function AllowDisallowIcmpOutboundPacketTooBig(Allow: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  FwProfile: INetFwProfile;
begin
  Result := S_OK;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
      FwMgr := INetFwMgr(FwMgrDisp);

      FwProfile := FwMgr.LocalPolicy.CurrentProfile;
      FwProfile.IcmpSettings.AllowOutboundPacketTooBig := Allow;
    finally
      FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

function IsIcmpTypeAllowed(IpVersion: NET_FW_IP_VERSION; LocalAddress: String;
  IcmpType: NET_FW_ICMP_TYPE; var Allowed: Boolean; var Restricted: Boolean): HRESULT;
var
  FwMgrDisp: IDispatch;
  FwMgr: INetFwMgr;
  TempAllowed: OleVariant;
  Temprestricted: OleVariant;
begin
  Result := S_OK;

  try
    FwMgrDisp := CreateOleObject(FW_MGR_CLASS_NAME);
    try
      FwMgr := INetFwMgr(FwMgrDisp);
      FwMgr.IsIcmpTypeAllowed(TOleEnum(IpVersion), LocalAddress, Byte(IcmpType), TempAllowed, TempRestricted);

      Allowed := Boolean(TempAllowed);
      Restricted := Boolean(TempRestricted);
    finally
      FwMgrDisp := Unassigned;
    end;
  except
    on E:EOleSysError do
      Result := E.ErrorCode;
  end;
end;

end.
