unit NetFwTypeLib_TLB;

// ************************************************************************ //
// WARNUNG                                                                    
// -------                                                                    
// Die in dieser Datei deklarierten Typen wurden aus Daten einer Typbibliothek
// generiert. Wenn diese Typbibliothek explizit oder indirekt (über eine     
// andere Typbibliothek) reimportiert wird oder wenn die Anweisung            
// 'Aktualisieren' im Typbibliotheks-Editor während des Bearbeitens der     
// Typbibliothek aktiviert ist, wird der Inhalt dieser Datei neu generiert und 
// alle manuell vorgenommenen Änderungen gehen verloren.                           
// ************************************************************************ //

// PASTLWTR : $Revision:   1.130.1.0.1.0.1.6  $
// Datei generiert am 20.05.2007 16:34:09 aus der unten beschriebenen Typbibliothek.

// ************************************************************************  //
// Type Lib: FirewallAPI.dll (1)
// LIBID: {58FBCF7C-E7A9-467C-80B3-FC65E8FCCA08}
// LCID: 0
// Helpfile: 
// DepndLst: 
//   (1) v2.0 stdole, (C:\WINDOWS\system32\stdole2.tlb)
//   (2) v4.0 StdVCL, (C:\WINDOWS\system32\stdvcl40.dll)
// Fehler
//   Hinweis: Element 'Type' von 'INetFwService' geändert in 'Type_'
//   Hinweis: Parameter 'Type' im INetFwService.Type geändert in 'Type_'
//   Hinweis: Element 'Type' von 'INetFwProfile' geändert in 'Type_'
//   Hinweis: Parameter 'Type' im INetFwProfile.Type geändert in 'Type_'
//   Hinweis: Parameter 'Type' im INetFwMgr.IsIcmpTypeAllowed geändert in 'Type_'
//   Hinweis: Element 'Type' von 'INetFwService' geändert in 'Type_'
//   Hinweis: Element 'Type' von 'INetFwProfile' geändert in 'Type_'
// ************************************************************************ //
// *************************************************************************//              
// HINWEIS:                                                                                   
// Von $IFDEF_LIVE_SERVER_AT_DESIGN_TIME überwachte Einträge, werden von  
// Eigenschaften verwendet, die Objekte zurückgeben, die explizit mit einen Funktionsaufruf  
// vor dem Zugriff über die Eigenschaft erzeugt werden müssen. Diese Einträge wurden deaktiviert,  
// um deren unbeabsichtigte Benutzung im Objektinspektor zu verhindern. Sie können sie  
// aktivieren, indem Sie LIVE_SERVER_AT_DESIGN_TIME definieren oder sie selektiv  
// aus den $IFDEF-Blöcken entfernen. Solche Einträge müssen jedoch programmseitig 
// mit einer Methode der geeigneten CoClass vor der Verwendung  
// erzeugt werden.                                                                 
{$TYPEDADDRESS OFF} // Unit muß ohne Typüberprüfung für Zeiger compiliert werden. 
{$WARN SYMBOL_PLATFORM OFF}
{$WRITEABLECONST ON}
{$VARPROPSETTER ON}
interface

uses Windows, ActiveX, Classes, Graphics, StdVCL, Variants;
  

// *********************************************************************//
// In dieser Typbibliothek deklarierte GUIDS . Es werden folgende         
// Präfixe verwendet:                                                     
//   Typbibliotheken     : LIBID_xxxx                                     
//   CoClasses           : CLASS_xxxx                                     
//   DISPInterfaces      : DIID_xxxx                                      
//   Nicht-DISP-Schnittstellen: IID_xxxx                                       
// *********************************************************************//
const
  // Haupt- und Nebenversionen der Typbibliothek
  NetFwTypeLibMajorVersion = 1;
  NetFwTypeLibMinorVersion = 0;

  LIBID_NetFwTypeLib: TGUID = '{58FBCF7C-E7A9-467C-80B3-FC65E8FCCA08}';

  IID_INetFwRemoteAdminSettings: TGUID = '{D4BECDDF-6F73-4A83-B832-9C66874CD20E}';
  IID_INetFwIcmpSettings: TGUID = '{A6207B2E-7CDD-426A-951E-5E1CBC5AFEAD}';
  IID_INetFwOpenPort: TGUID = '{E0483BA0-47FF-4D9C-A6D6-7741D0B195F7}';
  IID_INetFwOpenPorts: TGUID = '{C0E9D7FA-E07E-430A-B19A-090CE82D92E2}';
  IID_INetFwService: TGUID = '{79FD57C8-908E-4A36-9888-D5B3F0A444CF}';
  IID_INetFwServices: TGUID = '{79649BB4-903E-421B-94C9-79848E79F6EE}';
  IID_INetFwAuthorizedApplication: TGUID = '{B5E64FFA-C2C5-444E-A301-FB5E00018050}';
  IID_INetFwAuthorizedApplications: TGUID = '{644EFD52-CCF9-486C-97A2-39F352570B30}';
  IID_INetFwServiceRestriction: TGUID = '{8267BBE3-F890-491C-B7B6-2DB1EF0E5D2B}';
  IID_INetFwRules: TGUID = '{9C4C6277-5027-441E-AFAE-CA1F542DA009}';
  IID_INetFwRule: TGUID = '{AF230D27-BABA-4E42-ACED-F524F22CFCE2}';
  IID_INetFwProfile: TGUID = '{174A0DDA-E9F9-449D-993B-21AB667CA456}';
  IID_INetFwPolicy: TGUID = '{D46D2478-9AC9-4008-9DC7-5563CE5536CC}';
  IID_INetFwPolicy2: TGUID = '{98325047-C671-4174-8D81-DEFCD3F03186}';
  IID_INetFwMgr: TGUID = '{F7898AF5-CAC4-4632-A2EC-DA06E5111AF2}';

// *********************************************************************//
// Deklaration von in der Typbibliothek definierten Enumerationen         
// *********************************************************************//
// Konstanten für enum NET_FW_IP_VERSION_
type
  NET_FW_IP_VERSION_ = TOleEnum;
const
  NET_FW_IP_VERSION_V4 = $00000000;
  NET_FW_IP_VERSION_V6 = $00000001;
  NET_FW_IP_VERSION_ANY = $00000002;
  NET_FW_IP_VERSION_MAX = $00000003;

// Konstanten für enum NET_FW_SCOPE_
type
  NET_FW_SCOPE_ = TOleEnum;
const
  NET_FW_SCOPE_ALL = $00000000;
  NET_FW_SCOPE_LOCAL_SUBNET = $00000001;
  NET_FW_SCOPE_CUSTOM = $00000002;
  NET_FW_SCOPE_MAX = $00000003;

// Konstanten für enum NET_FW_IP_PROTOCOL_
type
  NET_FW_IP_PROTOCOL_ = TOleEnum;
const
  NET_FW_IP_PROTOCOL_TCP = $00000006;
  NET_FW_IP_PROTOCOL_UDP = $00000011;
  NET_FW_IP_PROTOCOL_ANY = $00000100;

// Konstanten für enum NET_FW_SERVICE_TYPE_
type
  NET_FW_SERVICE_TYPE_ = TOleEnum;
const
  NET_FW_SERVICE_FILE_AND_PRINT = $00000000;
  NET_FW_SERVICE_UPNP = $00000001;
  NET_FW_SERVICE_REMOTE_DESKTOP = $00000002;
  NET_FW_SERVICE_NONE = $00000003;
  NET_FW_SERVICE_TYPE_MAX = $00000004;

// Konstanten für enum NET_FW_RULE_DIRECTION_
type
  NET_FW_RULE_DIRECTION_ = TOleEnum;
const
  NET_FW_RULE_DIR_IN = $00000001;
  NET_FW_RULE_DIR_OUT = $00000002;
  NET_FW_RULE_DIR_MAX = $00000003;

// Konstanten für enum NET_FW_ACTION_
type
  NET_FW_ACTION_ = TOleEnum;
const
  NET_FW_ACTION_BLOCK = $00000000;
  NET_FW_ACTION_ALLOW = $00000001;
  NET_FW_ACTION_MAX = $00000002;

// Konstanten für enum NET_FW_PROFILE_TYPE_
type
  NET_FW_PROFILE_TYPE_ = TOleEnum;
const
  NET_FW_PROFILE_DOMAIN = $00000000;
  NET_FW_PROFILE_STANDARD = $00000001;
  NET_FW_PROFILE_CURRENT = $00000002;
  NET_FW_PROFILE_TYPE_MAX = $00000003;

// Konstanten für enum NET_FW_PROFILE_TYPE2_
type
  NET_FW_PROFILE_TYPE2_ = TOleEnum;
const
  NET_FW_PROFILE2_DOMAIN = $00000001;
  NET_FW_PROFILE2_PRIVATE = $00000002;
  NET_FW_PROFILE2_PUBLIC = $00000004;
  NET_FW_PROFILE2_ALL = $7FFFFFFF;

// Konstanten für enum NET_FW_MODIFY_STATE_
type
  NET_FW_MODIFY_STATE_ = TOleEnum;
const
  NET_FW_MODIFY_STATE_OK = $00000000;
  NET_FW_MODIFY_STATE_GP_OVERRIDE = $00000001;
  NET_FW_MODIFY_STATE_INBOUND_BLOCKED = $00000002;

type

// *********************************************************************//
// Forward-Deklaration von in der Typbibliothek definierten Typen         
// *********************************************************************//
  INetFwRemoteAdminSettings = interface;
  INetFwRemoteAdminSettingsDisp = dispinterface;
  INetFwIcmpSettings = interface;
  INetFwIcmpSettingsDisp = dispinterface;
  INetFwOpenPort = interface;
  INetFwOpenPortDisp = dispinterface;
  INetFwOpenPorts = interface;
  INetFwOpenPortsDisp = dispinterface;
  INetFwService = interface;
  INetFwServiceDisp = dispinterface;
  INetFwServices = interface;
  INetFwServicesDisp = dispinterface;
  INetFwAuthorizedApplication = interface;
  INetFwAuthorizedApplicationDisp = dispinterface;
  INetFwAuthorizedApplications = interface;
  INetFwAuthorizedApplicationsDisp = dispinterface;
  INetFwServiceRestriction = interface;
  INetFwServiceRestrictionDisp = dispinterface;
  INetFwRules = interface;
  INetFwRulesDisp = dispinterface;
  INetFwRule = interface;
  INetFwRuleDisp = dispinterface;
  INetFwProfile = interface;
  INetFwProfileDisp = dispinterface;
  INetFwPolicy = interface;
  INetFwPolicyDisp = dispinterface;
  INetFwPolicy2 = interface;
  INetFwPolicy2Disp = dispinterface;
  INetFwMgr = interface;
  INetFwMgrDisp = dispinterface;

// *********************************************************************//
// Schnittstelle: INetFwRemoteAdminSettings
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {D4BECDDF-6F73-4A83-B832-9C66874CD20E}
// *********************************************************************//
  INetFwRemoteAdminSettings = interface(IDispatch)
    ['{D4BECDDF-6F73-4A83-B832-9C66874CD20E}']
    function Get_IpVersion: NET_FW_IP_VERSION_; safecall;
    procedure Set_IpVersion(IpVersion: NET_FW_IP_VERSION_); safecall;
    function Get_Scope: NET_FW_SCOPE_; safecall;
    procedure Set_Scope(Scope: NET_FW_SCOPE_); safecall;
    function Get_RemoteAddresses: WideString; safecall;
    procedure Set_RemoteAddresses(const remoteAddrs: WideString); safecall;
    function Get_Enabled: WordBool; safecall;
    procedure Set_Enabled(Enabled: WordBool); safecall;
    property IpVersion: NET_FW_IP_VERSION_ read Get_IpVersion write Set_IpVersion;
    property Scope: NET_FW_SCOPE_ read Get_Scope write Set_Scope;
    property RemoteAddresses: WideString read Get_RemoteAddresses write Set_RemoteAddresses;
    property Enabled: WordBool read Get_Enabled write Set_Enabled;
  end;

// *********************************************************************//
// DispIntf:  INetFwRemoteAdminSettingsDisp
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {D4BECDDF-6F73-4A83-B832-9C66874CD20E}
// *********************************************************************//
  INetFwRemoteAdminSettingsDisp = dispinterface
    ['{D4BECDDF-6F73-4A83-B832-9C66874CD20E}']
    property IpVersion: NET_FW_IP_VERSION_ dispid 1;
    property Scope: NET_FW_SCOPE_ dispid 2;
    property RemoteAddresses: WideString dispid 3;
    property Enabled: WordBool dispid 4;
  end;

// *********************************************************************//
// Schnittstelle: INetFwIcmpSettings
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {A6207B2E-7CDD-426A-951E-5E1CBC5AFEAD}
// *********************************************************************//
  INetFwIcmpSettings = interface(IDispatch)
    ['{A6207B2E-7CDD-426A-951E-5E1CBC5AFEAD}']
    function Get_AllowOutboundDestinationUnreachable: WordBool; safecall;
    procedure Set_AllowOutboundDestinationUnreachable(allow: WordBool); safecall;
    function Get_AllowRedirect: WordBool; safecall;
    procedure Set_AllowRedirect(allow: WordBool); safecall;
    function Get_AllowInboundEchoRequest: WordBool; safecall;
    procedure Set_AllowInboundEchoRequest(allow: WordBool); safecall;
    function Get_AllowOutboundTimeExceeded: WordBool; safecall;
    procedure Set_AllowOutboundTimeExceeded(allow: WordBool); safecall;
    function Get_AllowOutboundParameterProblem: WordBool; safecall;
    procedure Set_AllowOutboundParameterProblem(allow: WordBool); safecall;
    function Get_AllowOutboundSourceQuench: WordBool; safecall;
    procedure Set_AllowOutboundSourceQuench(allow: WordBool); safecall;
    function Get_AllowInboundRouterRequest: WordBool; safecall;
    procedure Set_AllowInboundRouterRequest(allow: WordBool); safecall;
    function Get_AllowInboundTimestampRequest: WordBool; safecall;
    procedure Set_AllowInboundTimestampRequest(allow: WordBool); safecall;
    function Get_AllowInboundMaskRequest: WordBool; safecall;
    procedure Set_AllowInboundMaskRequest(allow: WordBool); safecall;
    function Get_AllowOutboundPacketTooBig: WordBool; safecall;
    procedure Set_AllowOutboundPacketTooBig(allow: WordBool); safecall;
    property AllowOutboundDestinationUnreachable: WordBool read Get_AllowOutboundDestinationUnreachable write Set_AllowOutboundDestinationUnreachable;
    property AllowRedirect: WordBool read Get_AllowRedirect write Set_AllowRedirect;
    property AllowInboundEchoRequest: WordBool read Get_AllowInboundEchoRequest write Set_AllowInboundEchoRequest;
    property AllowOutboundTimeExceeded: WordBool read Get_AllowOutboundTimeExceeded write Set_AllowOutboundTimeExceeded;
    property AllowOutboundParameterProblem: WordBool read Get_AllowOutboundParameterProblem write Set_AllowOutboundParameterProblem;
    property AllowOutboundSourceQuench: WordBool read Get_AllowOutboundSourceQuench write Set_AllowOutboundSourceQuench;
    property AllowInboundRouterRequest: WordBool read Get_AllowInboundRouterRequest write Set_AllowInboundRouterRequest;
    property AllowInboundTimestampRequest: WordBool read Get_AllowInboundTimestampRequest write Set_AllowInboundTimestampRequest;
    property AllowInboundMaskRequest: WordBool read Get_AllowInboundMaskRequest write Set_AllowInboundMaskRequest;
    property AllowOutboundPacketTooBig: WordBool read Get_AllowOutboundPacketTooBig write Set_AllowOutboundPacketTooBig;
  end;

// *********************************************************************//
// DispIntf:  INetFwIcmpSettingsDisp
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {A6207B2E-7CDD-426A-951E-5E1CBC5AFEAD}
// *********************************************************************//
  INetFwIcmpSettingsDisp = dispinterface
    ['{A6207B2E-7CDD-426A-951E-5E1CBC5AFEAD}']
    property AllowOutboundDestinationUnreachable: WordBool dispid 1;
    property AllowRedirect: WordBool dispid 2;
    property AllowInboundEchoRequest: WordBool dispid 3;
    property AllowOutboundTimeExceeded: WordBool dispid 4;
    property AllowOutboundParameterProblem: WordBool dispid 5;
    property AllowOutboundSourceQuench: WordBool dispid 6;
    property AllowInboundRouterRequest: WordBool dispid 7;
    property AllowInboundTimestampRequest: WordBool dispid 8;
    property AllowInboundMaskRequest: WordBool dispid 9;
    property AllowOutboundPacketTooBig: WordBool dispid 10;
  end;

// *********************************************************************//
// Schnittstelle: INetFwOpenPort
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {E0483BA0-47FF-4D9C-A6D6-7741D0B195F7}
// *********************************************************************//
  INetFwOpenPort = interface(IDispatch)
    ['{E0483BA0-47FF-4D9C-A6D6-7741D0B195F7}']
    function Get_Name: WideString; safecall;
    procedure Set_Name(const Name: WideString); safecall;
    function Get_IpVersion: NET_FW_IP_VERSION_; safecall;
    procedure Set_IpVersion(IpVersion: NET_FW_IP_VERSION_); safecall;
    function Get_Protocol: NET_FW_IP_PROTOCOL_; safecall;
    procedure Set_Protocol(ipProtocol: NET_FW_IP_PROTOCOL_); safecall;
    function Get_Port: Integer; safecall;
    procedure Set_Port(portNumber: Integer); safecall;
    function Get_Scope: NET_FW_SCOPE_; safecall;
    procedure Set_Scope(Scope: NET_FW_SCOPE_); safecall;
    function Get_RemoteAddresses: WideString; safecall;
    procedure Set_RemoteAddresses(const remoteAddrs: WideString); safecall;
    function Get_Enabled: WordBool; safecall;
    procedure Set_Enabled(Enabled: WordBool); safecall;
    function Get_BuiltIn: WordBool; safecall;
    property Name: WideString read Get_Name write Set_Name;
    property IpVersion: NET_FW_IP_VERSION_ read Get_IpVersion write Set_IpVersion;
    property Protocol: NET_FW_IP_PROTOCOL_ read Get_Protocol write Set_Protocol;
    property Port: Integer read Get_Port write Set_Port;
    property Scope: NET_FW_SCOPE_ read Get_Scope write Set_Scope;
    property RemoteAddresses: WideString read Get_RemoteAddresses write Set_RemoteAddresses;
    property Enabled: WordBool read Get_Enabled write Set_Enabled;
    property BuiltIn: WordBool read Get_BuiltIn;
  end;

// *********************************************************************//
// DispIntf:  INetFwOpenPortDisp
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {E0483BA0-47FF-4D9C-A6D6-7741D0B195F7}
// *********************************************************************//
  INetFwOpenPortDisp = dispinterface
    ['{E0483BA0-47FF-4D9C-A6D6-7741D0B195F7}']
    property Name: WideString dispid 1;
    property IpVersion: NET_FW_IP_VERSION_ dispid 2;
    property Protocol: NET_FW_IP_PROTOCOL_ dispid 3;
    property Port: Integer dispid 4;
    property Scope: NET_FW_SCOPE_ dispid 5;
    property RemoteAddresses: WideString dispid 6;
    property Enabled: WordBool dispid 7;
    property BuiltIn: WordBool readonly dispid 8;
  end;

// *********************************************************************//
// Schnittstelle: INetFwOpenPorts
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {C0E9D7FA-E07E-430A-B19A-090CE82D92E2}
// *********************************************************************//
  INetFwOpenPorts = interface(IDispatch)
    ['{C0E9D7FA-E07E-430A-B19A-090CE82D92E2}']
    function Get_Count: Integer; safecall;
    procedure Add(const Port: INetFwOpenPort); safecall;
    procedure Remove(portNumber: Integer; ipProtocol: NET_FW_IP_PROTOCOL_); safecall;
    function Item(portNumber: Integer; ipProtocol: NET_FW_IP_PROTOCOL_): INetFwOpenPort; safecall;
    function Get__NewEnum: IUnknown; safecall;
    property Count: Integer read Get_Count;
    property _NewEnum: IUnknown read Get__NewEnum;
  end;

// *********************************************************************//
// DispIntf:  INetFwOpenPortsDisp
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {C0E9D7FA-E07E-430A-B19A-090CE82D92E2}
// *********************************************************************//
  INetFwOpenPortsDisp = dispinterface
    ['{C0E9D7FA-E07E-430A-B19A-090CE82D92E2}']
    property Count: Integer readonly dispid 1;
    procedure Add(const Port: INetFwOpenPort); dispid 2;
    procedure Remove(portNumber: Integer; ipProtocol: NET_FW_IP_PROTOCOL_); dispid 3;
    function Item(portNumber: Integer; ipProtocol: NET_FW_IP_PROTOCOL_): INetFwOpenPort; dispid 4;
    property _NewEnum: IUnknown readonly dispid -4;
  end;

// *********************************************************************//
// Schnittstelle: INetFwService
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {79FD57C8-908E-4A36-9888-D5B3F0A444CF}
// *********************************************************************//
  INetFwService = interface(IDispatch)
    ['{79FD57C8-908E-4A36-9888-D5B3F0A444CF}']
    function Get_Name: WideString; safecall;
    function Get_Type_: NET_FW_SERVICE_TYPE_; safecall;
    function Get_Customized: WordBool; safecall;
    function Get_IpVersion: NET_FW_IP_VERSION_; safecall;
    procedure Set_IpVersion(IpVersion: NET_FW_IP_VERSION_); safecall;
    function Get_Scope: NET_FW_SCOPE_; safecall;
    procedure Set_Scope(Scope: NET_FW_SCOPE_); safecall;
    function Get_RemoteAddresses: WideString; safecall;
    procedure Set_RemoteAddresses(const remoteAddrs: WideString); safecall;
    function Get_Enabled: WordBool; safecall;
    procedure Set_Enabled(Enabled: WordBool); safecall;
    function Get_GloballyOpenPorts: INetFwOpenPorts; safecall;
    property Name: WideString read Get_Name;
    property Type_: NET_FW_SERVICE_TYPE_ read Get_Type_;
    property Customized: WordBool read Get_Customized;
    property IpVersion: NET_FW_IP_VERSION_ read Get_IpVersion write Set_IpVersion;
    property Scope: NET_FW_SCOPE_ read Get_Scope write Set_Scope;
    property RemoteAddresses: WideString read Get_RemoteAddresses write Set_RemoteAddresses;
    property Enabled: WordBool read Get_Enabled write Set_Enabled;
    property GloballyOpenPorts: INetFwOpenPorts read Get_GloballyOpenPorts;
  end;

// *********************************************************************//
// DispIntf:  INetFwServiceDisp
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {79FD57C8-908E-4A36-9888-D5B3F0A444CF}
// *********************************************************************//
  INetFwServiceDisp = dispinterface
    ['{79FD57C8-908E-4A36-9888-D5B3F0A444CF}']
    property Name: WideString readonly dispid 1;
    property Type_: NET_FW_SERVICE_TYPE_ readonly dispid 2;
    property Customized: WordBool readonly dispid 3;
    property IpVersion: NET_FW_IP_VERSION_ dispid 4;
    property Scope: NET_FW_SCOPE_ dispid 5;
    property RemoteAddresses: WideString dispid 6;
    property Enabled: WordBool dispid 7;
    property GloballyOpenPorts: INetFwOpenPorts readonly dispid 8;
  end;

// *********************************************************************//
// Schnittstelle: INetFwServices
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {79649BB4-903E-421B-94C9-79848E79F6EE}
// *********************************************************************//
  INetFwServices = interface(IDispatch)
    ['{79649BB4-903E-421B-94C9-79848E79F6EE}']
    function Get_Count: Integer; safecall;
    function Item(svcType: NET_FW_SERVICE_TYPE_): INetFwService; safecall;
    function Get__NewEnum: IUnknown; safecall;
    property Count: Integer read Get_Count;
    property _NewEnum: IUnknown read Get__NewEnum;
  end;

// *********************************************************************//
// DispIntf:  INetFwServicesDisp
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {79649BB4-903E-421B-94C9-79848E79F6EE}
// *********************************************************************//
  INetFwServicesDisp = dispinterface
    ['{79649BB4-903E-421B-94C9-79848E79F6EE}']
    property Count: Integer readonly dispid 1;
    function Item(svcType: NET_FW_SERVICE_TYPE_): INetFwService; dispid 2;
    property _NewEnum: IUnknown readonly dispid -4;
  end;

// *********************************************************************//
// Schnittstelle: INetFwAuthorizedApplication
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {B5E64FFA-C2C5-444E-A301-FB5E00018050}
// *********************************************************************//
  INetFwAuthorizedApplication = interface(IDispatch)
    ['{B5E64FFA-C2C5-444E-A301-FB5E00018050}']
    function Get_Name: WideString; safecall;
    procedure Set_Name(const Name: WideString); safecall;
    function Get_ProcessImageFileName: WideString; safecall;
    procedure Set_ProcessImageFileName(const imageFileName: WideString); safecall;
    function Get_IpVersion: NET_FW_IP_VERSION_; safecall;
    procedure Set_IpVersion(IpVersion: NET_FW_IP_VERSION_); safecall;
    function Get_Scope: NET_FW_SCOPE_; safecall;
    procedure Set_Scope(Scope: NET_FW_SCOPE_); safecall;
    function Get_RemoteAddresses: WideString; safecall;
    procedure Set_RemoteAddresses(const remoteAddrs: WideString); safecall;
    function Get_Enabled: WordBool; safecall;
    procedure Set_Enabled(Enabled: WordBool); safecall;
    property Name: WideString read Get_Name write Set_Name;
    property ProcessImageFileName: WideString read Get_ProcessImageFileName write Set_ProcessImageFileName;
    property IpVersion: NET_FW_IP_VERSION_ read Get_IpVersion write Set_IpVersion;
    property Scope: NET_FW_SCOPE_ read Get_Scope write Set_Scope;
    property RemoteAddresses: WideString read Get_RemoteAddresses write Set_RemoteAddresses;
    property Enabled: WordBool read Get_Enabled write Set_Enabled;
  end;

// *********************************************************************//
// DispIntf:  INetFwAuthorizedApplicationDisp
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {B5E64FFA-C2C5-444E-A301-FB5E00018050}
// *********************************************************************//
  INetFwAuthorizedApplicationDisp = dispinterface
    ['{B5E64FFA-C2C5-444E-A301-FB5E00018050}']
    property Name: WideString dispid 1;
    property ProcessImageFileName: WideString dispid 2;
    property IpVersion: NET_FW_IP_VERSION_ dispid 3;
    property Scope: NET_FW_SCOPE_ dispid 4;
    property RemoteAddresses: WideString dispid 5;
    property Enabled: WordBool dispid 6;
  end;

// *********************************************************************//
// Schnittstelle: INetFwAuthorizedApplications
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {644EFD52-CCF9-486C-97A2-39F352570B30}
// *********************************************************************//
  INetFwAuthorizedApplications = interface(IDispatch)
    ['{644EFD52-CCF9-486C-97A2-39F352570B30}']
    function Get_Count: Integer; safecall;
    procedure Add(const app: INetFwAuthorizedApplication); safecall;
    procedure Remove(const imageFileName: WideString); safecall;
    function Item(const imageFileName: WideString): INetFwAuthorizedApplication; safecall;
    function Get__NewEnum: IUnknown; safecall;
    property Count: Integer read Get_Count;
    property _NewEnum: IUnknown read Get__NewEnum;
  end;

// *********************************************************************//
// DispIntf:  INetFwAuthorizedApplicationsDisp
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {644EFD52-CCF9-486C-97A2-39F352570B30}
// *********************************************************************//
  INetFwAuthorizedApplicationsDisp = dispinterface
    ['{644EFD52-CCF9-486C-97A2-39F352570B30}']
    property Count: Integer readonly dispid 1;
    procedure Add(const app: INetFwAuthorizedApplication); dispid 2;
    procedure Remove(const imageFileName: WideString); dispid 3;
    function Item(const imageFileName: WideString): INetFwAuthorizedApplication; dispid 4;
    property _NewEnum: IUnknown readonly dispid -4;
  end;

// *********************************************************************//
// Schnittstelle: INetFwServiceRestriction
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {8267BBE3-F890-491C-B7B6-2DB1EF0E5D2B}
// *********************************************************************//
  INetFwServiceRestriction = interface(IDispatch)
    ['{8267BBE3-F890-491C-B7B6-2DB1EF0E5D2B}']
    procedure RestrictService(const serviceName: WideString; const appName: WideString; 
                              RestrictService: WordBool; serviceSidRestricted: WordBool); safecall;
    function ServiceRestricted(const serviceName: WideString; const appName: WideString): WordBool; safecall;
    function Get_Rules: INetFwRules; safecall;
    property Rules: INetFwRules read Get_Rules;
  end;

// *********************************************************************//
// DispIntf:  INetFwServiceRestrictionDisp
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {8267BBE3-F890-491C-B7B6-2DB1EF0E5D2B}
// *********************************************************************//
  INetFwServiceRestrictionDisp = dispinterface
    ['{8267BBE3-F890-491C-B7B6-2DB1EF0E5D2B}']
    procedure RestrictService(const serviceName: WideString; const appName: WideString; 
                              RestrictService: WordBool; serviceSidRestricted: WordBool); dispid 1;
    function ServiceRestricted(const serviceName: WideString; const appName: WideString): WordBool; dispid 2;
    property Rules: INetFwRules readonly dispid 3;
  end;

// *********************************************************************//
// Schnittstelle: INetFwRules
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {9C4C6277-5027-441E-AFAE-CA1F542DA009}
// *********************************************************************//
  INetFwRules = interface(IDispatch)
    ['{9C4C6277-5027-441E-AFAE-CA1F542DA009}']
    function Get_Count: Integer; safecall;
    procedure Add(const rule: INetFwRule); safecall;
    procedure Remove(const Name: WideString); safecall;
    function Item(const Name: WideString): INetFwRule; safecall;
    function Get__NewEnum: IUnknown; safecall;
    property Count: Integer read Get_Count;
    property _NewEnum: IUnknown read Get__NewEnum;
  end;

// *********************************************************************//
// DispIntf:  INetFwRulesDisp
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {9C4C6277-5027-441E-AFAE-CA1F542DA009}
// *********************************************************************//
  INetFwRulesDisp = dispinterface
    ['{9C4C6277-5027-441E-AFAE-CA1F542DA009}']
    property Count: Integer readonly dispid 1;
    procedure Add(const rule: INetFwRule); dispid 2;
    procedure Remove(const Name: WideString); dispid 3;
    function Item(const Name: WideString): INetFwRule; dispid 4;
    property _NewEnum: IUnknown readonly dispid -4;
  end;

// *********************************************************************//
// Schnittstelle: INetFwRule
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {AF230D27-BABA-4E42-ACED-F524F22CFCE2}
// *********************************************************************//
  INetFwRule = interface(IDispatch)
    ['{AF230D27-BABA-4E42-ACED-F524F22CFCE2}']
    function Get_Name: WideString; safecall;
    procedure Set_Name(const Name: WideString); safecall;
    function Get_Description: WideString; safecall;
    procedure Set_Description(const desc: WideString); safecall;
    function Get_ApplicationName: WideString; safecall;
    procedure Set_ApplicationName(const imageFileName: WideString); safecall;
    function Get_serviceName: WideString; safecall;
    procedure Set_serviceName(const serviceName: WideString); safecall;
    function Get_Protocol: Integer; safecall;
    procedure Set_Protocol(Protocol: Integer); safecall;
    function Get_LocalPorts: WideString; safecall;
    procedure Set_LocalPorts(const portNumbers: WideString); safecall;
    function Get_RemotePorts: WideString; safecall;
    procedure Set_RemotePorts(const portNumbers: WideString); safecall;
    function Get_LocalAddresses: WideString; safecall;
    procedure Set_LocalAddresses(const localAddrs: WideString); safecall;
    function Get_RemoteAddresses: WideString; safecall;
    procedure Set_RemoteAddresses(const remoteAddrs: WideString); safecall;
    function Get_IcmpTypesAndCodes: WideString; safecall;
    procedure Set_IcmpTypesAndCodes(const IcmpTypesAndCodes: WideString); safecall;
    function Get_Direction: NET_FW_RULE_DIRECTION_; safecall;
    procedure Set_Direction(dir: NET_FW_RULE_DIRECTION_); safecall;
    function Get_Interfaces: OleVariant; safecall;
    procedure Set_Interfaces(Interfaces: OleVariant); safecall;
    function Get_InterfaceTypes: WideString; safecall;
    procedure Set_InterfaceTypes(const InterfaceTypes: WideString); safecall;
    function Get_Enabled: WordBool; safecall;
    procedure Set_Enabled(Enabled: WordBool); safecall;
    function Get_Grouping: WideString; safecall;
    procedure Set_Grouping(const context: WideString); safecall;
    function Get_Profiles: Integer; safecall;
    procedure Set_Profiles(profileTypesBitmask: Integer); safecall;
    function Get_EdgeTraversal: WordBool; safecall;
    procedure Set_EdgeTraversal(Enabled: WordBool); safecall;
    function Get_Action: NET_FW_ACTION_; safecall;
    procedure Set_Action(Action: NET_FW_ACTION_); safecall;
    property Name: WideString read Get_Name write Set_Name;
    property Description: WideString read Get_Description write Set_Description;
    property ApplicationName: WideString read Get_ApplicationName write Set_ApplicationName;
    property serviceName: WideString read Get_serviceName write Set_serviceName;
    property Protocol: Integer read Get_Protocol write Set_Protocol;
    property LocalPorts: WideString read Get_LocalPorts write Set_LocalPorts;
    property RemotePorts: WideString read Get_RemotePorts write Set_RemotePorts;
    property LocalAddresses: WideString read Get_LocalAddresses write Set_LocalAddresses;
    property RemoteAddresses: WideString read Get_RemoteAddresses write Set_RemoteAddresses;
    property IcmpTypesAndCodes: WideString read Get_IcmpTypesAndCodes write Set_IcmpTypesAndCodes;
    property Direction: NET_FW_RULE_DIRECTION_ read Get_Direction write Set_Direction;
    property Interfaces: OleVariant read Get_Interfaces write Set_Interfaces;
    property InterfaceTypes: WideString read Get_InterfaceTypes write Set_InterfaceTypes;
    property Enabled: WordBool read Get_Enabled write Set_Enabled;
    property Grouping: WideString read Get_Grouping write Set_Grouping;
    property Profiles: Integer read Get_Profiles write Set_Profiles;
    property EdgeTraversal: WordBool read Get_EdgeTraversal write Set_EdgeTraversal;
    property Action: NET_FW_ACTION_ read Get_Action write Set_Action;
  end;

// *********************************************************************//
// DispIntf:  INetFwRuleDisp
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {AF230D27-BABA-4E42-ACED-F524F22CFCE2}
// *********************************************************************//
  INetFwRuleDisp = dispinterface
    ['{AF230D27-BABA-4E42-ACED-F524F22CFCE2}']
    property Name: WideString dispid 1;
    property Description: WideString dispid 2;
    property ApplicationName: WideString dispid 3;
    property serviceName: WideString dispid 4;
    property Protocol: Integer dispid 5;
    property LocalPorts: WideString dispid 6;
    property RemotePorts: WideString dispid 7;
    property LocalAddresses: WideString dispid 8;
    property RemoteAddresses: WideString dispid 9;
    property IcmpTypesAndCodes: WideString dispid 10;
    property Direction: NET_FW_RULE_DIRECTION_ dispid 11;
    property Interfaces: OleVariant dispid 12;
    property InterfaceTypes: WideString dispid 13;
    property Enabled: WordBool dispid 14;
    property Grouping: WideString dispid 15;
    property Profiles: Integer dispid 16;
    property EdgeTraversal: WordBool dispid 17;
    property Action: NET_FW_ACTION_ dispid 18;
  end;

// *********************************************************************//
// Schnittstelle: INetFwProfile
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {174A0DDA-E9F9-449D-993B-21AB667CA456}
// *********************************************************************//
  INetFwProfile = interface(IDispatch)
    ['{174A0DDA-E9F9-449D-993B-21AB667CA456}']
    function Get_Type_: NET_FW_PROFILE_TYPE_; safecall;
    function Get_FirewallEnabled: WordBool; safecall;
    procedure Set_FirewallEnabled(Enabled: WordBool); safecall;
    function Get_ExceptionsNotAllowed: WordBool; safecall;
    procedure Set_ExceptionsNotAllowed(notAllowed: WordBool); safecall;
    function Get_NotificationsDisabled: WordBool; safecall;
    procedure Set_NotificationsDisabled(disabled: WordBool); safecall;
    function Get_UnicastResponsesToMulticastBroadcastDisabled: WordBool; safecall;
    procedure Set_UnicastResponsesToMulticastBroadcastDisabled(disabled: WordBool); safecall;
    function Get_RemoteAdminSettings: INetFwRemoteAdminSettings; safecall;
    function Get_IcmpSettings: INetFwIcmpSettings; safecall;
    function Get_GloballyOpenPorts: INetFwOpenPorts; safecall;
    function Get_Services: INetFwServices; safecall;
    function Get_AuthorizedApplications: INetFwAuthorizedApplications; safecall;
    property Type_: NET_FW_PROFILE_TYPE_ read Get_Type_;
    property FirewallEnabled: WordBool read Get_FirewallEnabled write Set_FirewallEnabled;
    property ExceptionsNotAllowed: WordBool read Get_ExceptionsNotAllowed write Set_ExceptionsNotAllowed;
    property NotificationsDisabled: WordBool read Get_NotificationsDisabled write Set_NotificationsDisabled;
    property UnicastResponsesToMulticastBroadcastDisabled: WordBool read Get_UnicastResponsesToMulticastBroadcastDisabled write Set_UnicastResponsesToMulticastBroadcastDisabled;
    property RemoteAdminSettings: INetFwRemoteAdminSettings read Get_RemoteAdminSettings;
    property IcmpSettings: INetFwIcmpSettings read Get_IcmpSettings;
    property GloballyOpenPorts: INetFwOpenPorts read Get_GloballyOpenPorts;
    property Services: INetFwServices read Get_Services;
    property AuthorizedApplications: INetFwAuthorizedApplications read Get_AuthorizedApplications;
  end;

// *********************************************************************//
// DispIntf:  INetFwProfileDisp
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {174A0DDA-E9F9-449D-993B-21AB667CA456}
// *********************************************************************//
  INetFwProfileDisp = dispinterface
    ['{174A0DDA-E9F9-449D-993B-21AB667CA456}']
    property Type_: NET_FW_PROFILE_TYPE_ readonly dispid 1;
    property FirewallEnabled: WordBool dispid 2;
    property ExceptionsNotAllowed: WordBool dispid 3;
    property NotificationsDisabled: WordBool dispid 4;
    property UnicastResponsesToMulticastBroadcastDisabled: WordBool dispid 5;
    property RemoteAdminSettings: INetFwRemoteAdminSettings readonly dispid 6;
    property IcmpSettings: INetFwIcmpSettings readonly dispid 7;
    property GloballyOpenPorts: INetFwOpenPorts readonly dispid 8;
    property Services: INetFwServices readonly dispid 9;
    property AuthorizedApplications: INetFwAuthorizedApplications readonly dispid 10;
  end;

// *********************************************************************//
// Schnittstelle: INetFwPolicy
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {D46D2478-9AC9-4008-9DC7-5563CE5536CC}
// *********************************************************************//
  INetFwPolicy = interface(IDispatch)
    ['{D46D2478-9AC9-4008-9DC7-5563CE5536CC}']
    function Get_CurrentProfile: INetFwProfile; safecall;
    function GetProfileByType(profileType: NET_FW_PROFILE_TYPE_): INetFwProfile; safecall;
    property CurrentProfile: INetFwProfile read Get_CurrentProfile;
  end;

// *********************************************************************//
// DispIntf:  INetFwPolicyDisp
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {D46D2478-9AC9-4008-9DC7-5563CE5536CC}
// *********************************************************************//
  INetFwPolicyDisp = dispinterface
    ['{D46D2478-9AC9-4008-9DC7-5563CE5536CC}']
    property CurrentProfile: INetFwProfile readonly dispid 1;
    function GetProfileByType(profileType: NET_FW_PROFILE_TYPE_): INetFwProfile; dispid 2;
  end;

// *********************************************************************//
// Schnittstelle: INetFwPolicy2
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {98325047-C671-4174-8D81-DEFCD3F03186}
// *********************************************************************//
  INetFwPolicy2 = interface(IDispatch)
    ['{98325047-C671-4174-8D81-DEFCD3F03186}']
    function Get_CurrentProfileTypes: Integer; safecall;
    function Get_FirewallEnabled(profileType: NET_FW_PROFILE_TYPE2_): WordBool; safecall;
    procedure Set_FirewallEnabled(profileType: NET_FW_PROFILE_TYPE2_; Enabled: WordBool); safecall;
    function Get_ExcludedInterfaces(profileType: NET_FW_PROFILE_TYPE2_): OleVariant; safecall;
    procedure Set_ExcludedInterfaces(profileType: NET_FW_PROFILE_TYPE2_; Interfaces: OleVariant); safecall;
    function Get_BlockAllInboundTraffic(profileType: NET_FW_PROFILE_TYPE2_): WordBool; safecall;
    procedure Set_BlockAllInboundTraffic(profileType: NET_FW_PROFILE_TYPE2_; Block: WordBool); safecall;
    function Get_NotificationsDisabled(profileType: NET_FW_PROFILE_TYPE2_): WordBool; safecall;
    procedure Set_NotificationsDisabled(profileType: NET_FW_PROFILE_TYPE2_; disabled: WordBool); safecall;
    function Get_UnicastResponsesToMulticastBroadcastDisabled(profileType: NET_FW_PROFILE_TYPE2_): WordBool; safecall;
    procedure Set_UnicastResponsesToMulticastBroadcastDisabled(profileType: NET_FW_PROFILE_TYPE2_; 
                                                               disabled: WordBool); safecall;
    function Get_Rules: INetFwRules; safecall;
    function Get_ServiceRestriction: INetFwServiceRestriction; safecall;
    procedure EnableRuleGroup(profileTypesBitmask: Integer; const group: WideString; 
                              enable: WordBool); safecall;
    function IsRuleGroupEnabled(profileTypesBitmask: Integer; const group: WideString): WordBool; safecall;
    procedure RestoreLocalFirewallDefaults; safecall;
    function Get_DefaultInboundAction(profileType: NET_FW_PROFILE_TYPE2_): NET_FW_ACTION_; safecall;
    procedure Set_DefaultInboundAction(profileType: NET_FW_PROFILE_TYPE2_; Action: NET_FW_ACTION_); safecall;
    function Get_DefaultOutboundAction(profileType: NET_FW_PROFILE_TYPE2_): NET_FW_ACTION_; safecall;
    procedure Set_DefaultOutboundAction(profileType: NET_FW_PROFILE_TYPE2_; Action: NET_FW_ACTION_); safecall;
    function Get_IsRuleGroupCurrentlyEnabled(const group: WideString): WordBool; safecall;
    function Get_LocalPolicyModifyState: NET_FW_MODIFY_STATE_; safecall;
    property CurrentProfileTypes: Integer read Get_CurrentProfileTypes;
    property FirewallEnabled[profileType: NET_FW_PROFILE_TYPE2_]: WordBool read Get_FirewallEnabled write Set_FirewallEnabled;
    property ExcludedInterfaces[profileType: NET_FW_PROFILE_TYPE2_]: OleVariant read Get_ExcludedInterfaces write Set_ExcludedInterfaces;
    property BlockAllInboundTraffic[profileType: NET_FW_PROFILE_TYPE2_]: WordBool read Get_BlockAllInboundTraffic write Set_BlockAllInboundTraffic;
    property NotificationsDisabled[profileType: NET_FW_PROFILE_TYPE2_]: WordBool read Get_NotificationsDisabled write Set_NotificationsDisabled;
    property UnicastResponsesToMulticastBroadcastDisabled[profileType: NET_FW_PROFILE_TYPE2_]: WordBool read Get_UnicastResponsesToMulticastBroadcastDisabled write Set_UnicastResponsesToMulticastBroadcastDisabled;
    property Rules: INetFwRules read Get_Rules;
    property ServiceRestriction: INetFwServiceRestriction read Get_ServiceRestriction;
    property DefaultInboundAction[profileType: NET_FW_PROFILE_TYPE2_]: NET_FW_ACTION_ read Get_DefaultInboundAction write Set_DefaultInboundAction;
    property DefaultOutboundAction[profileType: NET_FW_PROFILE_TYPE2_]: NET_FW_ACTION_ read Get_DefaultOutboundAction write Set_DefaultOutboundAction;
    property IsRuleGroupCurrentlyEnabled[const group: WideString]: WordBool read Get_IsRuleGroupCurrentlyEnabled;
    property LocalPolicyModifyState: NET_FW_MODIFY_STATE_ read Get_LocalPolicyModifyState;
  end;

// *********************************************************************//
// DispIntf:  INetFwPolicy2Disp
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {98325047-C671-4174-8D81-DEFCD3F03186}
// *********************************************************************//
  INetFwPolicy2Disp = dispinterface
    ['{98325047-C671-4174-8D81-DEFCD3F03186}']
    property CurrentProfileTypes: Integer readonly dispid 1;
    property FirewallEnabled[profileType: NET_FW_PROFILE_TYPE2_]: WordBool dispid 2;
    property ExcludedInterfaces[profileType: NET_FW_PROFILE_TYPE2_]: OleVariant dispid 3;
    property BlockAllInboundTraffic[profileType: NET_FW_PROFILE_TYPE2_]: WordBool dispid 4;
    property NotificationsDisabled[profileType: NET_FW_PROFILE_TYPE2_]: WordBool dispid 5;
    property UnicastResponsesToMulticastBroadcastDisabled[profileType: NET_FW_PROFILE_TYPE2_]: WordBool dispid 6;
    property Rules: INetFwRules readonly dispid 7;
    property ServiceRestriction: INetFwServiceRestriction readonly dispid 8;
    procedure EnableRuleGroup(profileTypesBitmask: Integer; const group: WideString; 
                              enable: WordBool); dispid 9;
    function IsRuleGroupEnabled(profileTypesBitmask: Integer; const group: WideString): WordBool; dispid 10;
    procedure RestoreLocalFirewallDefaults; dispid 11;
    property DefaultInboundAction[profileType: NET_FW_PROFILE_TYPE2_]: NET_FW_ACTION_ dispid 12;
    property DefaultOutboundAction[profileType: NET_FW_PROFILE_TYPE2_]: NET_FW_ACTION_ dispid 13;
    property IsRuleGroupCurrentlyEnabled[const group: WideString]: WordBool readonly dispid 14;
    property LocalPolicyModifyState: NET_FW_MODIFY_STATE_ readonly dispid 15;
  end;

// *********************************************************************//
// Schnittstelle: INetFwMgr
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {F7898AF5-CAC4-4632-A2EC-DA06E5111AF2}
// *********************************************************************//
  INetFwMgr = interface(IDispatch)
    ['{F7898AF5-CAC4-4632-A2EC-DA06E5111AF2}']
    function Get_LocalPolicy: INetFwPolicy; safecall;
    function Get_CurrentProfileType: NET_FW_PROFILE_TYPE_; safecall;
    procedure RestoreDefaults; safecall;
    procedure IsPortAllowed(const imageFileName: WideString; IpVersion: NET_FW_IP_VERSION_; 
                            portNumber: Integer; const localAddress: WideString; 
                            ipProtocol: NET_FW_IP_PROTOCOL_; out allowed: OleVariant; 
                            out restricted: OleVariant); safecall;
    procedure IsIcmpTypeAllowed(IpVersion: NET_FW_IP_VERSION_; const localAddress: WideString; 
                                Type_: Byte; out allowed: OleVariant; out restricted: OleVariant); safecall;
    property LocalPolicy: INetFwPolicy read Get_LocalPolicy;
    property CurrentProfileType: NET_FW_PROFILE_TYPE_ read Get_CurrentProfileType;
  end;

// *********************************************************************//
// DispIntf:  INetFwMgrDisp
// Flags:     (4416) Dual OleAutomation Dispatchable
// GUID:      {F7898AF5-CAC4-4632-A2EC-DA06E5111AF2}
// *********************************************************************//
  INetFwMgrDisp = dispinterface
    ['{F7898AF5-CAC4-4632-A2EC-DA06E5111AF2}']
    property LocalPolicy: INetFwPolicy readonly dispid 1;
    property CurrentProfileType: NET_FW_PROFILE_TYPE_ readonly dispid 2;
    procedure RestoreDefaults; dispid 3;
    procedure IsPortAllowed(const imageFileName: WideString; IpVersion: NET_FW_IP_VERSION_; 
                            portNumber: Integer; const localAddress: WideString; 
                            ipProtocol: NET_FW_IP_PROTOCOL_; out allowed: OleVariant; 
                            out restricted: OleVariant); dispid 4;
    procedure IsIcmpTypeAllowed(IpVersion: NET_FW_IP_VERSION_; const localAddress: WideString; 
                                Type_: Byte; out allowed: OleVariant; out restricted: OleVariant); dispid 5;
  end;

implementation

uses ComObj;

end.
