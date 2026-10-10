package search

import "strings"

// Each line is a group of words that mean the same thing, so people can
// search in Portuguese or English ("capacitor" finds "Condensador"). Words
// are written without accents; plurals are handled by Stem.
const synonymGroups = `
resistencia resistor
condensador capacitor cap
indutor inductor bobina coil
diodo diode
transistor transistor
cristal crystal
oscilador oscillator
conector connector ficha plug
terminal crimp
interruptor switch
botao button
rele relay
fusivel fuse
amplificador amplifier ampop opamp
conversor converter
regulador regulator
memoria memory
filtro filter
blindagem shield
termopar thermocouple
sensor sensor
pressao pressure
temperatura temperature
humidade humidity
bateria battery
placa board plate pcb
modulo module
porta gate
logica logic
parafuso screw bolt
porca nut
anilha washer
mola spring
fio wire thread
cabo cable
corda rope cord
mangueira hose
valvula valve
manometro gauge manometer
tubo tube pipe
tanque tank
camara chamber
combustao combustion
paraquedas parachute chute
jaula cage
janela window
perfil profile extrusion
aluminio aluminium aluminum
aco steel
madeira wood
cola glue adhesive adesivo
ferramenta tool
equipamento equipment
maquina machine
costura sewing
roldana pulley
guincho winch
mosquetao carabiner
suporte support mount bracket holder
gaveta drawer
caixa box
prateleira shelf
armario cabinet
sala room
parede wall
polvora powder
cartucho cartridge
ignicao ignition
perigoso hazardous
foguete rocket
preto preta black
vermelho vermelha red
amarelo amarela yellow
azul blue
verde green
branco branca white
rosa pink
`

// synonyms maps a word's stem to the other words of its groups.
var synonyms = func() map[string][]string {
	m := map[string][]string{}
	for _, line := range strings.Split(synonymGroups, "\n") {
		words := strings.Fields(line)
		for _, w := range words {
			k := Stem(w)
			for _, o := range words {
				if o != w {
					m[k] = append(m[k], o)
				}
			}
		}
	}
	return m
}()

// translations returns the other-language words for w. Words of 4 letters or
// more also expand while still being typed ("capac" -> "condensador").
func translations(w string) []string {
	ws := Stem(w)
	if len(ws) < 4 {
		return synonyms[ws]
	}
	var out []string
	for k, v := range synonyms {
		if strings.HasPrefix(k, ws) {
			out = append(out, v...)
		}
	}
	return out
}
