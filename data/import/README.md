# Imported inventory (October 2026)

Import files converted from the team's two spreadsheets (`Inventário.xlsx` and
`Inventário Sala FADEUP.xlsx`). Neither spreadsheet was complete or final.

| File | From | Rows |
|---|---|---|
| `eletronica.csv` | electronics stock: boxes 1–3, drawers 1–63 | 244 |
| `sala-fadeup.csv` | Sala FADEUP: Ground, Estruturas, Recuperação & Payload sheets | 139 |

Import with **Import CSV** in the app, or `POST /api/import/items?commit=1`.
Importing a file twice **adds the quantities twice**, so import each file only once
per database.

## How the data was mapped

- **Electronics**
  - Location: `Núcleo PST/Caixa N/Gaveta M`. When the boxes find their place in the new
    room, move each box into it; the drawers and their contents move along.
  - Category: `Eletrónica/<sheet>`. Parts that were in the wrong sheet were moved to the
    right category (see below).
  - Name: the type plus the value or part number. Passives also get a package
    ("Resistência 10k Ω", package 0603). A part with a specific part number never merges with
    a generic one of the same value.
  - Owner: the team.
- **Sala FADEUP**
  - Location: `Sala FADEUP/<department sheet>`. The sheets don't say where each thing is.
  - Owner: INVICTUS Ground, Structures, or Recovery & Payload.
  - Boxes with mixed contents ("caixa de cartão — porcas e parafusos") became locations
    holding one "not counted" item.
- **Quantities**
  - `+++`, `muitos` and `demasiadas` became "lots, not counted".
  - `~10` became an estimate.
  - `9+1`, `25+2+3` and `12 (3x4)` were added up.
  - `5 m` uses metres as the unit.
- **Reservations:** notes that say what something is for ("Para o ring da jaula", "Para as
  Radax", "Avionics") became reservations of the whole amount for that department, with the
  purpose as the note.
- **Broken units:** quick-links and swivel links marked "Não funcionais" were imported as broken
  units.

## To check (in the spreadsheets or the room)

- **Resistências, drawer 1 (45 pcs):** Excel turned the value into a date. Imported as
  "Resistência (valor por confirmar)".
- **Parts moved out of the Diodos sheet:**
  - LD1117V33, NCR321UX, AP64501SP-13 and 74AHC1G08SE-7 now go under IC.
  - ZVN4206GTC now goes under Transistor.
- **Missing or zero quantities:**
  - E28-2G4M27S (LoRa) has quantity 0, so the item exists with no stock.
  - Black powder has no quantity. The item "Pólvora negra" exists with no stock; weigh it and
    add it.
  - "Disco de paraquedas" has no quantity; imported as 1.
- **Unclear entries:**
  - "CHAVES" (2, THT drawer 47) has no reference; confirm what they are.
  - Connector 1722536112 had no drawer; assumed "Cima da gaveta 2" like the row before.
  - Shock cord codes (a*, a (2), c (3)…) are in the serial number. "(2)" may mean 2 units.
- **Possible duplicates:**
  - Estruturas: combustion chamber and tank are listed twice; imported 1 each.
  - R&P: "Paraquedas drogue 0.855" (4) and "Paraquedas drogue D0.855 E" (4) may be the same
    parachutes; both were imported.
- **Not imported:**
  - Estruturas: the "Antigos 2 / Grandes 2 / Pequenos 3" block (no header saying what it is).
  - R&P: one row with only a quantity (1).
- **Not reserved:** Ground's M4/M10 screws and nuts "para os pés do tripé / main body" are not
  counted, so they weren't reserved. The purpose is in their description.
