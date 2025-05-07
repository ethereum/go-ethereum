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

The original code is ServiceControl.pas, released April 16, 2007.

The initial developer of the original code is Rainer Budde (http://www.speed-soft.de).

SimpleSC - NSIS Service Control Plugin is written, published and maintaned by
Rainer Budde (rainer@speed-soft.de).
}
unit ServiceControl;

interface

uses
  Windows, SysUtils, WinSvc;

  function InstallService(ServiceName, DisplayName: String; ServiceType: DWORD; StartType: DWORD; BinaryPathName: String; Dependencies: String; Username: String; Password: String): Integer;
  function RemoveService(ServiceName: String): Integer;
  function GetServiceName(DisplayName: String; var Name: String): Integer;
  function GetServiceDisplayName(ServiceName: String; var Name: String): Integer;
  function GetServiceStatus(ServiceName: String; var Status: DWORD): Integer;
  function GetServiceBinaryPath(ServiceName: String; var BinaryPath: String): Integer;
  function GetServiceStartType(ServiceName: String; var StartType: DWORD): Integer;
  function GetServiceDescription(ServiceName: String; var Description: String): Integer;
  function GetServiceLogon(ServiceName: String; var Username: String): Integer;
  function SetServiceStartType(ServiceName: String; StartType: DWORD): Integer;
  function SetServiceDescription(ServiceName: String; Description: String): Integer;
  function SetServiceLogon(ServiceName: String; Username: String; Password: String): Integer;
  function SetServiceBinaryPath(ServiceName: String; BinaryPath: String): Integer;
  function ServiceIsRunning(ServiceName: String; var IsRunning: Boolean): Integer;
  function ServiceIsStopped(ServiceName: String; var IsStopped: Boolean): Integer;
  function ServiceIsPaused(ServiceName: String; var IsPaused: Boolean): Integer;
  function StartService(ServiceName: String; ServiceArguments: String): Integer;
  function StopService(ServiceName: String): Integer;
  function PauseService(ServiceName: String): Integer;
  function ContinueService(ServiceName: String): Integer;
  function RestartService(ServiceName: String; ServiceArguments: String): Integer;
  function ExistsService(ServiceName: String): Integer;
  function GetErrorMessage(ErrorCode: Integer): String;

implementation

function WaitForStatus(ServiceName: String; Status: DWord): Integer;
var
  CurrentStatus: DWord;
  StatusResult: Integer;
  StatusReached: Boolean;
  TimeOutReached: Boolean;
  StartTickCount: Cardinal;
const
  STATUS_TIMEOUT = 30000;
  WAIT_TIMEOUT = 250;
begin
  Result := 0;

  StatusReached := False;
  TimeOutReached := False;

  StartTickCount := GetTickCount;

  while not StatusReached and not TimeOutReached do
  begin
    StatusResult := GetServiceStatus(ServiceName, CurrentStatus);

    if StatusResult = 0 then
    begin
      if Status = CurrentStatus then
        StatusReached := True
      else
        Sleep(WAIT_TIMEOUT);
    end
    else
      Result := StatusResult;

    if (StartTickCount + STATUS_TIMEOUT) < GetTickCount then
    begin
      TimeOutReached := True;
      Result := ERROR_SERVICE_REQUEST_TIMEOUT;
    end;
  end;

end;

function ExistsService(ServiceName: String): Integer;
var
  ManagerHandle: SC_HANDLE;
  ServiceHandle: SC_HANDLE;
begin
  Result := 0;

  ManagerHandle := OpenSCManager('', nil, SC_MANAGER_CONNECT);

  if ManagerHandle > 0 then
  begin
    ServiceHandle := OpenService(ManagerHandle, PChar(ServiceName), SERVICE_QUERY_CONFIG);

    if ServiceHandle > 0 then
      CloseServiceHandle(ServiceHandle)
    else
      Result := System.GetLastError;

    CloseServiceHandle(ManagerHandle);
  end
  else
    Result := System.GetLastError;
end;

function StartService(ServiceName: String; ServiceArguments: String): Integer;
type
  TArguments = Array of PChar;
var
  ManagerHandle: SC_HANDLE;
  ServiceHandle: SC_HANDLE;
  ServiceArgVectors: TArguments;
  NumServiceArgs: DWORD;
const
  ArgDelimitterQuote: String = '"';
  ArgDelimitterWhiteSpace: String = ' ';

  procedure GetServiceArguments(ServiceArguments: String; var NumServiceArgs: DWORD; var ServiceArgVectors: TArguments);
  var
    Param: String;
    Split: Boolean;
    Quoted: Boolean;
    CharIsDelimitter: Boolean;
  begin
    ServiceArgVectors := nil;
    NumServiceArgs := 0;

    Quoted := False;

    while Length(ServiceArguments) > 0 do
    begin
      Split := False;
      CharIsDelimitter := False;

      if ServiceArguments[1] = ' ' then
        if not Quoted then
        begin
          CharIsDelimitter := True;
          Split := True;
        end;

      if ServiceArguments[1] = '"' then
      begin
        Quoted := not Quoted;
        CharIsDelimitter := True;

        if not Quoted then
          Split := True;
      end;

      if not CharIsDelimitter then
        Param := Param + ServiceArguments[1];

      if Split or (Length(ServiceArguments) = 1) then
      begin
        SetLength(ServiceArgVectors, Length(ServiceArgVectors) + 1);
        GetMem(ServiceArgVectors[Length(ServiceArgVectors) -1], Length(Param) + 1);
        StrPCopy(ServiceArgVectors[Length(ServiceArgVectors) -1], Param);

        Param := '';

        Delete(ServiceArguments, 1, 1);
        ServiceArguments := Trim(ServiceArguments);
      end
      else
        Delete(ServiceArguments, 1, 1);

    end;

    if Length(ServiceArgVectors) > 0 then
      NumServiceArgs := Length(ServiceArgVectors);
  end;

  procedure FreeServiceArguments(ServiceArgVectors: TArguments);
  var
    i: Integer;
  begin
    if Length(ServiceArgVectors) > 0 then
      for i := 0 to Length(ServiceArgVectors) -1 do
        FreeMem(ServiceArgVectors[i]);
  end;

begin
  ManagerHandle := OpenSCManager('', nil, SC_MANAGER_CONNECT);

  if ManagerHandle > 0 then
  begin
    ServiceHandle := OpenService(ManagerHandle, PChar(ServiceName), SERVICE_START);

    if ServiceHandle > 0 then
    begin
      GetServiceArguments(ServiceArguments, NumServiceArgs, ServiceArgVectors);

      if WinSvc.StartService(ServiceHandle, NumServiceArgs, ServiceArgVectors[0]) then
        Result := WaitForStatus(ServiceName, SERVICE_RUNNING)
      else
        Result := System.GetLastError;

      FreeServiceArguments(ServiceArgVectors);

      CloseServiceHandle(ServiceHandle);
    end
    else
      Result := System.GetLastError;


    CloseServiceHandle(ManagerHandle);
  end
  else
    Result := System.GetLastError;
end;

function StopService(ServiceName: String): Integer;
var
  ManagerHandle: SC_HANDLE;
  ServiceHandle: SC_HANDLE;
  ServiceStatus: TServiceStatus;
  Dependencies: PEnumServiceStatus;
  BytesNeeded: Cardinal;
  ServicesReturned: Cardinal;
  ServicesEnumerated: Boolean;
  EnumerationSuccess: Boolean;
  i: Cardinal;
begin
  Result := 0;

  BytesNeeded := 0;
  ServicesReturned := 0;

  Dependencies := nil;
  ServicesEnumerated := False;

  ManagerHandle := OpenSCManager('', nil, SC_MANAGER_CONNECT or SC_MANAGER_ENUMERATE_SERVICE);

  if ManagerHandle > 0 then
  begin
    ServiceHandle := OpenService(ManagerHandle, PChar(ServiceName), SERVICE_STOP or SERVICE_ENUMERATE_DEPENDENTS);

    if ServiceHandle > 0 then
    begin
      if not EnumDependentServices(ServiceHandle, SERVICE_ACTIVE, Dependencies^, 0, BytesNeeded, ServicesReturned) then
      begin
        ServicesEnumerated := True;
        GetMem(Dependencies, BytesNeeded);

        EnumerationSuccess := EnumDependentServices(ServiceHandle, SERVICE_ACTIVE, Dependencies^, BytesNeeded, BytesNeeded, ServicesReturned);

        if EnumerationSuccess and (ServicesReturned > 0) then
        begin
          for i := 1 to ServicesReturned do
          begin
            Result := StopService(Dependencies.lpServiceName);

            if Result <> 0 then
              Break;

            Inc(Dependencies);
          end;
        end
        else
          Result := System.GetLastError;
      end;

      if (ServicesEnumerated and (Result = 0)) or not ServicesEnumerated then
      begin
        if ControlService(ServiceHandle, SERVICE_CONTROL_STOP, ServiceStatus) then
          Result := WaitForStatus(ServiceName, SERVICE_STOPPED)
        else
          Result := System.GetLastError
      end;

      CloseServiceHandle(ServiceHandle);
    end
    else
      Result := System.GetLastError;

    CloseServiceHandle(ManagerHandle);
  end
  else
    Result := System.GetLastError;
end;

function PauseService(ServiceName: String): Integer;
var
  ManagerHandle: SC_HANDLE;
  ServiceHandle: SC_HANDLE;
  ServiceStatus: TServiceStatus;
begin
  ManagerHandle := OpenSCManager('', nil, SC_MANAGER_CONNECT);

  if ManagerHandle > 0 then
  begin
    ServiceHandle := OpenService(ManagerHandle, PChar(ServiceName), SERVICE_PAUSE_CONTINUE);

    if ServiceHandle > 0 then
    begin

      if ControlService(ServiceHandle, SERVICE_CONTROL_PAUSE, ServiceStatus) then
        Result := WaitForStatus(ServiceName, SERVICE_PAUSED)
      else
        Result := System.GetLastError;

      CloseServiceHandle(ServiceHandle);
    end
    else
      Result := System.GetLastError;

    CloseServiceHandle(ManagerHandle);
  end
  else
    Result := System.GetLastError;
end;

function ContinueService(ServiceName: String): Integer;
var
  ManagerHandle: SC_HANDLE;
  ServiceHandle: SC_HANDLE;
  ServiceStatus: TServiceStatus;
begin
  ManagerHandle := OpenSCManager('', nil, SC_MANAGER_CONNECT);

  if ManagerHandle > 0 then
  begin
    ServiceHandle := OpenService(ManagerHandle, PChar(ServiceName), SERVICE_PAUSE_CONTINUE);

    if ServiceHandle > 0 then
    begin

      if ControlService(ServiceHandle, SERVICE_CONTROL_CONTINUE, ServiceStatus) then
        Result := WaitForStatus(ServiceName, SERVICE_RUNNING)
      else
        Result := System.GetLastError;

      CloseServiceHandle(ServiceHandle);
    end
    else
      Result := System.GetLastError;

    CloseServiceHandle(ManagerHandle);
  end
  else
    Result := System.GetLastError;
end;

function GetServiceName(DisplayName: String; var Name: String): Integer;
var
  ManagerHandle: SC_HANDLE;
  ServiceName: PChar;
  ServiceBuffer: Cardinal;
begin
  Result := 0;

  ServiceBuffer := 255;
  ServiceName := StrAlloc(ServiceBuffer+1);

  ManagerHandle := OpenSCManager('', nil, SC_MANAGER_CONNECT);

  if ManagerHandle > 0 then
  begin
    if WinSvc.GetServiceKeyName(ManagerHandle, PChar(DisplayName), ServiceName, ServiceBuffer) then
      Name := ServiceName
    else
      Result := System.GetLastError;

    CloseServiceHandle(ManagerHandle);
  end
  else
    Result := System.GetLastError;
end;

function GetServiceDisplayName(ServiceName: String; var Name: String): Integer;
var
  ManagerHandle: SC_HANDLE;
  DisplayName: PChar;
  ServiceBuffer: Cardinal;
begin
  Result := 0;

  ServiceBuffer := 255;
  DisplayName := StrAlloc(ServiceBuffer+1);

  ManagerHandle := OpenSCManager('', nil, SC_MANAGER_CONNECT);

  if ManagerHandle > 0 then
  begin
    if WinSvc.GetServiceDisplayName(ManagerHandle, PChar(ServiceName), DisplayName, ServiceBuffer) then
      Name := DisplayName
    else
      Result := System.GetLastError;

    CloseServiceHandle(ManagerHandle);
  end
  else
    Result := System.GetLastError;
end;

function GetServiceStatus(ServiceName: String; var Status: DWORD): Integer;
var
  ManagerHandle: SC_HANDLE;
  ServiceHandle: SC_HANDLE;
  ServiceStatus: TServiceStatus;
begin
  Result := 0;

  ManagerHandle := OpenSCManager('', nil, SC_MANAGER_CONNECT);

  if ManagerHandle > 0 then
  begin
    ServiceHandle := OpenService(ManagerHandle, PChar(ServiceName), SERVICE_QUERY_STATUS);

    if ServiceHandle > 0 then
    begin
      if QueryServiceStatus(ServiceHandle, ServiceStatus) then
        Status := ServiceStatus.dwCurrentState
      else
        Result := System.GetLastError;

      CloseServiceHandle(ServiceHandle);
    end
    else
      Result := System.GetLastError;

    CloseServiceHandle(ManagerHandle);
  end
  else
    Result := System.GetLastError;
end;

function GetServiceBinaryPath(ServiceName: String; var BinaryPath: String): Integer;
var
  ManagerHandle: SC_HANDLE;
  ServiceHandle: SC_HANDLE;
  BytesNeeded: DWORD;
  ServiceConfig: PQueryServiceConfig;
begin
  Result := 0;
  ServiceConfig := nil;

  ManagerHandle := OpenSCManager('', nil, SC_MANAGER_CONNECT);

  if ManagerHandle > 0 then
  begin
    ServiceHandle := OpenService(ManagerHandle, PChar(ServiceName), SERVICE_QUERY_CONFIG);

    if ServiceHandle > 0 then
    begin

      if not QueryServiceConfig(ServiceHandle, ServiceConfig, 0, BytesNeeded) and (System.GetLastError = ERROR_INSUFFICIENT_BUFFER) then
      begin
        GetMem(ServiceConfig, BytesNeeded);

        if QueryServiceConfig(ServiceHandle, ServiceConfig, BytesNeeded, BytesNeeded) then
          BinaryPath := ServiceConfig^.lpBinaryPathName
        else
          Result := System.GetLastError;

        FreeMem(ServiceConfig);
      end
      else
        Result := System.GetLastError;

      CloseServiceHandle(ServiceHandle);
    end
    else
      Result := System.GetLastError;

    CloseServiceHandle(ManagerHandle);
  end
  else
    Result := System.GetLastError;
end;

function GetServiceStartType(ServiceName: String; var StartType: DWORD): Integer;
var
  ManagerHandle: SC_HANDLE;
  ServiceHandle: SC_HANDLE;
  BytesNeeded: DWORD;
  ServiceConfig: PQueryServiceConfig;
begin
  Result := 0;
  ServiceConfig := nil;

  ManagerHandle := OpenSCManager('', nil, SC_MANAGER_CONNECT);

  if ManagerHandle > 0 then
  begin
    ServiceHandle := OpenService(ManagerHandle, PChar(ServiceName), SERVICE_QUERY_CONFIG);

    if ServiceHandle > 0 then
    begin

      if not QueryServiceConfig(ServiceHandle, ServiceConfig, 0, BytesNeeded) and (System.GetLastError = ERROR_INSUFFICIENT_BUFFER) then
      begin
        GetMem(ServiceConfig, BytesNeeded);

        if QueryServiceConfig(ServiceHandle, ServiceConfig, BytesNeeded, BytesNeeded) then
          StartType := ServiceConfig^.dwStartType
        else
          Result := System.GetLastError;

        FreeMem(ServiceConfig);
      end
      else
        Result := System.GetLastError;

      CloseServiceHandle(ServiceHandle);
    end
    else
      Result := System.GetLastError;

    CloseServiceHandle(ManagerHandle);
  end
  else
    Result := System.GetLastError;
end;

function GetServiceDescription(ServiceName: String; var Description: String): Integer;
const
  SERVICE_CONFIG_DESCRIPTION = 1;
type
  TServiceDescription = record
    lpDescription: PAnsiChar;
  end;
  PServiceDescription = ^TServiceDescription;
var
  QueryServiceConfig2: function(hService: SC_HANDLE; dwInfoLevel: DWORD; pBuffer: Pointer; cbBufSize: DWORD; var cbBytesNeeded: Cardinal): BOOL; stdcall;
  ManagerHandle: SC_HANDLE;
  ServiceHandle: SC_HANDLE;
  LockHandle: SC_LOCK;
  ServiceDescription: PServiceDescription;
  BytesNeeded: Cardinal;
begin
  Result := 0;

  ManagerHandle := OpenSCManager('', nil, SC_MANAGER_LOCK);

  if ManagerHandle > 0 then
  begin
    ServiceHandle := OpenService(ManagerHandle, PChar(ServiceName), SERVICE_QUERY_CONFIG);

    if ServiceHandle > 0 then
    begin
      LockHandle := LockServiceDatabase(ManagerHandle);

      if LockHandle <> nil then
      begin
        @QueryServiceConfig2 := GetProcAddress(GetModuleHandle(advapi32), 'QueryServiceConfig2A');

        if Assigned(@QueryServiceConfig2) then
        begin

          if not QueryServiceConfig2(ServiceHandle, SERVICE_CONFIG_DESCRIPTION, nil, 0, BytesNeeded) and (System.GetLastError = ERROR_INSUFFICIENT_BUFFER) then
          begin
            GetMem(ServiceDescription, BytesNeeded);

            if QueryServiceConfig2(ServiceHandle, SERVICE_CONFIG_DESCRIPTION, ServiceDescription, BytesNeeded, BytesNeeded) then
              Description := ServiceDescription.lpDescription
            else
              Result := System.GetLastError;

            FreeMem(ServiceDescription);
          end
          else
            Result := System.GetLastError;

        end
        else
          Result := System.GetLastError;

        UnlockServiceDatabase(LockHandle);
      end
      else
        Result := System.GetLastError;

      CloseServiceHandle(ServiceHandle);
    end
    else
      Result := System.GetLastError;

    CloseServiceHandle(ManagerHandle);
  end
  else
    Result := System.GetLastError;
end;

function GetServiceLogon(ServiceName: String; var Username: String): Integer;
var
  ManagerHandle: SC_HANDLE;
  ServiceHandle: SC_HANDLE;
  BytesNeeded: DWORD;
  ServiceConfig: PQueryServiceConfig;
begin
  Result := 0;
  ServiceConfig := nil;

  ManagerHandle := OpenSCManager('', nil, SC_MANAGER_CONNECT);

  if ManagerHandle > 0 then
  begin
    ServiceHandle := OpenService(ManagerHandle, PChar(ServiceName), SERVICE_QUERY_CONFIG);

    if ServiceHandle > 0 then
    begin

      if not QueryServiceConfig(ServiceHandle, ServiceConfig, 0, BytesNeeded) and (System.GetLastError = ERROR_INSUFFICIENT_BUFFER) then
      begin
        GetMem(ServiceConfig, BytesNeeded);

        if QueryServiceConfig(ServiceHandle, ServiceConfig, BytesNeeded, BytesNeeded) then
          Username := ServiceConfig^.lpServiceStartName
        else
          Result := System.GetLastError;

        FreeMem(ServiceConfig);
      end
      else
        Result := System.GetLastError;

      CloseServiceHandle(ServiceHandle);
    end
    else
      Result := System.GetLastError;

    CloseServiceHandle(ManagerHandle);
  end
  else
    Result := System.GetLastError;
end;

function SetServiceDescription(ServiceName: String; Description: String): Integer;
const
  SERVICE_CONFIG_DESCRIPTION = 1;
var
  ChangeServiceConfig2: function(hService: SC_HANDLE; dwInfoLevel: DWORD; lpInfo: Pointer): BOOL; stdcall;
  ManagerHandle: SC_HANDLE;
  ServiceHandle: SC_HANDLE;
  LockHandle: SC_LOCK;
begin
  Result := 0;

  ManagerHandle := OpenSCManager('', nil, SC_MANAGER_LOCK);

  if ManagerHandle > 0 then
  begin
    ServiceHandle := OpenService(ManagerHandle, PChar(ServiceName), SERVICE_CHANGE_CONFIG);

    if ServiceHandle > 0 then
    begin
      LockHandle := LockServiceDatabase(ManagerHandle);

      if LockHandle <> nil then
      begin
        @ChangeServiceConfig2 := GetProcAddress(GetModuleHandle(advapi32), 'ChangeServiceConfig2A');

        if Assigned(@ChangeServiceConfig2) then
        begin
          if not ChangeServiceConfig2(ServiceHandle, SERVICE_CONFIG_DESCRIPTION, @Description) then
            Result := System.GetLastError;
        end
        else
          Result := System.GetLastError;

        UnlockServiceDatabase(LockHandle);
      end
      else
        Result := System.GetLastError;

      CloseServiceHandle(ServiceHandle);
    end
    else
      Result := System.GetLastError;

    CloseServiceHandle(ManagerHandle);
  end
  else
    Result := System.GetLastError;
end;

function SetServiceStartType(ServiceName: String; StartType: DWORD): Integer;
var
  ManagerHandle: SC_HANDLE;
  ServiceHandle: SC_HANDLE;
  LockHandle: SC_LOCK;
begin
  Result := 0;

  ManagerHandle := OpenSCManager('', nil, SC_MANAGER_LOCK);

  if ManagerHandle > 0 then
  begin
    ServiceHandle := OpenService(ManagerHandle, PChar(ServiceName), SERVICE_CHANGE_CONFIG);

    if ServiceHandle > 0 then
    begin
      LockHandle := LockServiceDatabase(ManagerHandle);

      if LockHandle <> nil then
      begin
        if not ChangeServiceConfig(ServiceHandle, SERVICE_NO_CHANGE, StartType, SERVICE_NO_CHANGE, nil, nil, nil, nil, nil, nil, nil) then
          Result := System.GetLastError;

        UnlockServiceDatabase(LockHandle);
      end
      else
        Result := System.GetLastError;

      CloseServiceHandle(ServiceHandle);
    end
    else
      Result := System.GetLastError;

    CloseServiceHandle(ManagerHandle);
  end
  else
    Result := System.GetLastError;
end;

function SetServiceLogon(ServiceName: String; Username: String; Password: String): Integer;
var
  ManagerHandle: SC_HANDLE;
  ServiceHandle: SC_HANDLE;
  LockHandle: SC_LOCK;
begin
  Result := 0;

  ManagerHandle := OpenSCManager('', nil, SC_MANAGER_LOCK);

  if Pos('\', Username) = 0 then
    Username := '.\' + Username;

  if ManagerHandle > 0 then
  begin
    ServiceHandle := OpenService(ManagerHandle, PChar(ServiceName), SERVICE_CHANGE_CONFIG);

    if ServiceHandle > 0 then
    begin
      LockHandle := LockServiceDatabase(ManagerHandle);

      if LockHandle <> nil then
      begin
        if not ChangeServiceConfig(ServiceHandle, SERVICE_NO_CHANGE, SERVICE_NO_CHANGE, SERVICE_NO_CHANGE, nil, nil, nil, nil, PChar(Username), PChar(Password), nil) then
          Result := System.GetLastError;

        UnlockServiceDatabase(LockHandle);
      end
      else
        Result := System.GetLastError;

      CloseServiceHandle(ServiceHandle);
    end
    else
      Result := System.GetLastError;

    CloseServiceHandle(ManagerHandle);
  end
  else
    Result := System.GetLastError;
end;

function SetServiceBinaryPath(ServiceName: String; BinaryPath: String): Integer;
var
  ManagerHandle: SC_HANDLE;
  ServiceHandle: SC_HANDLE;
  LockHandle: SC_LOCK;
begin
  Result := 0;

  ManagerHandle := OpenSCManager('', nil, SC_MANAGER_LOCK);

  if ManagerHandle > 0 then
  begin
    ServiceHandle := OpenService(ManagerHandle, PChar(ServiceName), SERVICE_CHANGE_CONFIG);

    if ServiceHandle > 0 then
    begin
      LockHandle := LockServiceDatabase(ManagerHandle);

      if LockHandle <> nil then
      begin
        if not ChangeServiceConfig(ServiceHandle, SERVICE_NO_CHANGE, SERVICE_NO_CHANGE, SERVICE_NO_CHANGE, PChar(BinaryPath), nil, nil, nil, nil, nil, nil) then
          Result := System.GetLastError;

        UnlockServiceDatabase(LockHandle);
      end
      else
        Result := System.GetLastError;

      CloseServiceHandle(ServiceHandle);
    end
    else
      Result := System.GetLastError;

    CloseServiceHandle(ManagerHandle);
  end
  else
    Result := System.GetLastError;
end;

function ServiceIsRunning(ServiceName: String; var IsRunning: Boolean): Integer;
var
  Status: DWORD;
begin
  Result := GetServiceStatus(ServiceName, Status);

  if Result = 0 then
    IsRunning := Status = SERVICE_RUNNING
  else
    IsRunning := False;
end;

function ServiceIsStopped(ServiceName: String; var IsStopped: Boolean): Integer;
var
  Status: DWORD;
begin
  Result := GetServiceStatus(ServiceName, Status);

  if Result = 0 then
    IsStopped := Status = SERVICE_STOPPED
  else
    IsStopped := False;
end;

function ServiceIsPaused(ServiceName: String; var IsPaused: Boolean): Integer;
var
  Status: DWORD;
begin
  Result := GetServiceStatus(ServiceName, Status);

  if Result = 0 then
    IsPaused := Status = SERVICE_PAUSED
  else
    IsPaused := False;
end;

function RestartService(ServiceName: String; ServiceArguments: String): Integer;
begin
  Result := StopService(ServiceName);

  if Result = 0 then
    Result := StartService(ServiceName, ServiceArguments);
end;

function InstallService(ServiceName, DisplayName: String; ServiceType: DWORD;
  StartType: DWORD; BinaryPathName: String; Dependencies: String;
  Username: String; Password: String): Integer;
var
  ManagerHandle: SC_HANDLE;
  ServiceHandle: SC_HANDLE;
  PDependencies: PChar;
  PUsername: PChar;
  PPassword: PChar;
const
  ReplaceDelimitter: String = '/';

  function Replace(Value: String): String;
  begin
    while Pos(ReplaceDelimitter, Value) <> 0 do
    begin
      Result := Result + Copy(Value, 1, Pos(ReplaceDelimitter, Value) -1) + Chr(0);
      Delete(Value, 1, Pos(ReplaceDelimitter, Value));
    end;

    Result := Result + Value + Chr(0) + Chr(0);
  end;

begin
  Result := 0;
  
  if Dependencies = '' then
    PDependencies := nil
  else
    PDependencies := PChar(Replace(Dependencies));

  if UserName = '' then
    PUsername := nil
  else
    PUsername := PChar(Username);

  if Password = '' then
    PPassword := nil
  else
    PPassword := PChar(Password);

  ManagerHandle := OpenSCManager('', nil, SC_MANAGER_ALL_ACCESS);

  if ManagerHandle > 0 then
  begin
    ServiceHandle := CreateService(ManagerHandle,
                                   PChar(ServiceName),
                                   PChar(DisplayName),
                                   SERVICE_START or SERVICE_QUERY_STATUS or _DELETE,
                                   ServiceType,
                                   StartType,
                                   SERVICE_ERROR_NORMAL,
                                   PChar(BinaryPathName),
                                   nil,
                                   nil,
                                   PDependencies,
                                   PUsername,
                                   PPassword);

    if ServiceHandle <> 0 then
      CloseServiceHandle(ServiceHandle)
    else
      Result := System.GetLastError;

    CloseServiceHandle(ManagerHandle);
  end
  else
    Result := System.GetLastError;
end;

function RemoveService(ServiceName: String): Integer;
var
  ManagerHandle: SC_HANDLE;
  ServiceHandle: SC_HANDLE;
  LockHandle: SC_LOCK;
  IsStopped: Boolean;
  Deleted: Boolean;
begin
  IsStopped := False;

  Result := ServiceIsStopped(ServiceName, IsStopped);

  if Result = 0 then
    if not IsStopped then
      Result := StopService(ServiceName);

  if Result = 0 then
  begin
    ManagerHandle := OpenSCManager('', nil, SC_MANAGER_ALL_ACCESS);

    if ManagerHandle > 0 then
    begin
      ServiceHandle := OpenService(ManagerHandle, PChar(ServiceName), SERVICE_ALL_ACCESS);

      if ServiceHandle > 0 then
      begin
        LockHandle := LockServiceDatabase(ManagerHandle);

        if LockHandle <> nil then
        begin
          Deleted := DeleteService(ServiceHandle);

          if not Deleted then
            Result := System.GetLastError;

          UnlockServiceDatabase(LockHandle);
        end
        else
          Result := System.GetLastError;

        CloseServiceHandle(ServiceHandle);
      end
      else
        Result := System.GetLastError;

      CloseServiceHandle(ManagerHandle);
    end
    else
      Result := System.GetLastError;
  end;
end;

function GetErrorMessage(ErrorCode: Integer): String;
begin
  Result := SysErrorMessage(ErrorCode);
end;

end.
