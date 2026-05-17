Attribute VB_Name = "SafeSocket"
' -----------------------------------------------------------------------------------------------
' Safe Socket VBA Binding
' -----------------------------------------------------------------------------------------------
' This module provides access to the libsafesocket shared library from VBA.
' -----------------------------------------------------------------------------------------------

Option Explicit

#If Mac Then
    Private Const LIB_PATH As String = "libsafesocket.dylib"
#Else
    Private Const LIB_PATH As String = "libsafesocket.dll"
#End If

' Bridge API Declarations
' -----------------------------------------------------------------------------------------------

Public Declare PtrSafe Function SafeSocket_GetSocketError Lib LIB_PATH () As LongPtr
Public Declare PtrSafe Sub SafeSocket_FreeString Lib LIB_PATH (ByVal ptr As LongPtr)

Public Declare PtrSafe Function SafeSocket_Create Lib LIB_PATH (ByVal profileName As String, ByVal address As String, ByVal publicIP As String, ByVal socketType As String, ByVal autoConnect As Long) As Long
Public Declare PtrSafe Function SafeSocket_CreateExtended Lib LIB_PATH (ByVal profileName As String, ByVal address As String, ByVal publicIP As String, ByVal socketType As String, ByVal handshakeTimeoutMs As Long, ByVal deadlineMs As Long, ByVal heartbeatIntervalMs As Long, ByVal autoConnect As Long) As Long

Public Declare PtrSafe Function SafeSocket_Open Lib LIB_PATH (ByVal handle As Long) As Long
Public Declare PtrSafe Function SafeSocket_Close Lib LIB_PATH (ByVal handle As Long) As Long
Public Declare PtrSafe Function SafeSocket_Send Lib LIB_PATH (ByVal handle As Long, ByRef data As Byte, ByVal length As Long) As Long
Public Declare PtrSafe Function SafeSocket_Receive Lib LIB_PATH (ByVal handle As Long, ByRef buffer As Byte, ByVal maxLength As Long) As Long
Public Declare PtrSafe Function SafeSocket_Listen Lib LIB_PATH (ByVal handle As Long) As Long
Public Declare PtrSafe Function SafeSocket_Accept Lib LIB_PATH (ByVal handle As Long) As Long
Public Declare PtrSafe Function SafeSocket_SetIdleTimeout Lib LIB_PATH (ByVal handle As Long, ByVal seconds As Double) As Long
Public Declare PtrSafe Function SafeSocket_SetDeadline Lib LIB_PATH (ByVal handle As Long, ByVal seconds As Double) As Long

' -----------------------------------------------------------------------------------------------
' High-level API Example
' -----------------------------------------------------------------------------------------------

Public Sub DemoSafeSocket()
    Dim handle As Long
    ' Create a client socket
    ' SafeSocket_Create(profileName, address, publicIP, socketType, autoConnect)
    handle = SafeSocket_Create("demo", "localhost:8080", "", "client", 1)
    
    If handle = -1 Then
        Debug.Print "Failed to create socket"
        Exit Sub
    End If
    
    ' Send some data
    Dim msg As String
    msg = "Hello from VBA"
    Dim data() As Byte
    data = StrConv(msg, vbFromUnicode)
    
    If SafeSocket_Send(handle, data(0), UBound(data) + 1) = -1 Then
        Debug.Print "Send failed"
    End If
    
    ' Receive data
    Dim rxBuffer(0 To 1023) As Byte
    Dim n As Long
    n = SafeSocket_Receive(handle, rxBuffer(0), 1024)
    If n > 0 Then
        Debug.Print "Received " & n & " bytes"
    End If
    
    ' Clean up
    SafeSocket_Close handle
End Sub
