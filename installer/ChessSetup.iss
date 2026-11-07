; Chess v2.0.0 - standalone setup. Build: ISCC.exe ChessSetup.iss
; (c) 2026 Ironite
#define GameName    "Chess"
#define Bundle      "Chess"
#define Exe         "Chess.exe"
#define GameVersion "2.0.0"
#define AppGuid     "{{9B3E2C71-5D84-4A6F-B0C2-7E1A93D4F628}"
#define Art         "chess"
#define Tagline     "Six opponents at the Ravenscroft Library, from beginner to master, a friend on the same computer, or a friend online."
#define SaveWhat    "unfinished game, record and settings"

; Let Chess host online games without a firewall prompt.
[Run]
Filename: "{sys}\netsh.exe"; Parameters: "advfirewall firewall add rule name=""Chess"" dir=in action=allow program=""{app}\Games\{#Bundle}\{#Exe}"" enable=yes"; Flags: runhidden; StatusMsg: "Letting Chess host online games..."

[UninstallRun]
Filename: "{sys}\netsh.exe"; Parameters: "advfirewall firewall delete rule name=""Chess"""; Flags: runhidden; RunOnceId: "ChessFirewall"

#include "GameSetup.iss"
