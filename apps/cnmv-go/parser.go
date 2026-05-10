package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"time"
)

// XML structs mirror only the fields we persist. The CNMV XSD has more.

type registroClase struct {
	NumeroClase       int64  `xml:"NumeroClase"`
	DenominacionClase string `xml:"DenominacionClase"`
	ISIN              string `xml:"ISIN"`
}

type registroCompartimento struct {
	NumeroCompartimento       int64           `xml:"NumeroCompartimento"`
	DenominacionCompartimento string          `xml:"DenominacionCompartimento"`
	Clase                     []registroClase `xml:"Clase"`
}

type registroEntidad struct {
	Tipo           string                  `xml:"Tipo"`
	NumeroRegistro int64                   `xml:"NumeroRegistro"`
	Denominacion   string                  `xml:"Denominacion"`
	ETF            string                  `xml:"ETF"`
	Compartimento  []registroCompartimento `xml:"Compartimento"`
	Gestora        struct {
		DenominacionGestora string `xml:"DenominacionGestora"`
		GrupoGestora        struct {
			DenominacionGrupoGestora string `xml:"DenominacionGrupoGestora"`
		} `xml:"GrupoGestora"`
	} `xml:"Gestora"`
	Depositario struct {
		DenominacionDepositario string `xml:"DenominacionDepositario"`
		GrupoDepositario        struct {
			DenominacionGrupoDepositario string `xml:"DenominacionGrupoDepositario"`
		} `xml:"GrupoDepositario"`
	} `xml:"Depositario"`
}

// ParseRegistro streams the FONDREGISTRO file one <Entidad> at a time.
// Returns one Fund per (Entidad, Compartimento, Clase) and the period
// (YYYYMM) declared in <FechaDatos>.
func ParseRegistro(xmlData []byte) ([]Fund, string, error) {
	dec := xml.NewDecoder(bytes.NewReader(xmlData))
	var period string
	var out []Fund

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", fmt.Errorf("parse registro: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "FechaDatos":
			if err := dec.DecodeElement(&period, &se); err != nil {
				return nil, "", err
			}
		case "Entidad":
			var ent registroEntidad
			if err := dec.DecodeElement(&ent, &se); err != nil {
				return nil, "", err
			}
			isETF := ent.ETF == "SI"
			for _, comp := range ent.Compartimento {
				for _, cls := range comp.Clase {
					if cls.ISIN == "" {
						continue
					}
					out = append(out, Fund{
						ISIN:                      cls.ISIN,
						Tipo:                      ent.Tipo,
						NumeroRegistro:            ent.NumeroRegistro,
						NumeroCompartimento:       comp.NumeroCompartimento,
						NumeroClase:               cls.NumeroClase,
						Denominacion:              ent.Denominacion,
						DenominacionCompartimento: comp.DenominacionCompartimento,
						DenominacionClase:         cls.DenominacionClase,
						IsETF:                     isETF,
						GestoraNombre:             ent.Gestora.DenominacionGestora,
						GestoraGrupo:              ent.Gestora.GrupoGestora.DenominacionGrupoGestora,
						DepositarioNombre:         ent.Depositario.DenominacionDepositario,
						DepositarioGrupo:          ent.Depositario.GrupoDepositario.DenominacionGrupoDepositario,
						Currency:                  "EUR",
					})
				}
			}
		}
	}
	return out, period, nil
}

type mensClase struct {
	NumeroClase      int64  `xml:"NumeroClase"`
	ISIN             string `xml:"ISIN"`
	VLDiario         daysFloat
	ParticipesDiario daysInt
	PatrimonioDiario daysFloat
}

// UnmarshalXML for mensClase routes the VLDiario/ParticipesDiario/Patrimonio
// children to the appropriate days-array decoder. Done manually so we don't
// declare 31 fields per array.
func (m *mensClase) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "NumeroClase":
				_ = d.DecodeElement(&m.NumeroClase, &t)
			case "ISIN":
				_ = d.DecodeElement(&m.ISIN, &t)
			case "VLDiario":
				if err := m.VLDiario.decode(d, t, "VL_Dia"); err != nil {
					return err
				}
			case "ParticipesDiario":
				if err := m.ParticipesDiario.decode(d, t, "Participes_Dia"); err != nil {
					return err
				}
			case "PatrimonioDiario":
				if err := m.PatrimonioDiario.decode(d, t, "Patrimonio_Dia"); err != nil {
					return err
				}
			default:
				if err := d.Skip(); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if t.Name.Local == start.Name.Local {
				return nil
			}
		}
	}
}

type daysFloat [32]float64

func (a *daysFloat) decode(d *xml.Decoder, start xml.StartElement, prefix string) error {
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			day, ok := dayFromTag(t.Name.Local, prefix)
			if !ok {
				_ = d.Skip()
				continue
			}
			var raw string
			if err := d.DecodeElement(&raw, &t); err != nil {
				return err
			}
			v, _ := strconv.ParseFloat(raw, 64)
			a[day] = v
		case xml.EndElement:
			if t.Name.Local == start.Name.Local {
				return nil
			}
		}
	}
}

type daysInt [32]int64

func (a *daysInt) decode(d *xml.Decoder, start xml.StartElement, prefix string) error {
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			day, ok := dayFromTag(t.Name.Local, prefix)
			if !ok {
				_ = d.Skip()
				continue
			}
			var raw string
			if err := d.DecodeElement(&raw, &t); err != nil {
				return err
			}
			v, _ := strconv.ParseInt(raw, 10, 64)
			a[day] = v
		case xml.EndElement:
			if t.Name.Local == start.Name.Local {
				return nil
			}
		}
	}
}

func dayFromTag(tag, prefix string) (int, bool) {
	if len(tag) <= len(prefix) || tag[:len(prefix)] != prefix {
		return 0, false
	}
	n, err := strconv.Atoi(tag[len(prefix):])
	if err != nil || n < 1 || n > 31 {
		return 0, false
	}
	return n, true
}

// ParseMens streams FONDMENS one <Clase> element at a time and emits one
// NAVObservation per valid (ISIN, day). Days with VL=0 (non-trading or
// missing) and days outside the calendar month (Feb 30/31, Apr 31, ...) are
// dropped.
func ParseMens(xmlData []byte) ([]NAVObservation, error) {
	dec := xml.NewDecoder(bytes.NewReader(xmlData))
	var year, month int
	var out []NAVObservation

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse mens: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "FechaDatos":
			var fd string
			if err := dec.DecodeElement(&fd, &se); err != nil {
				return nil, err
			}
			if len(fd) == 6 {
				year, _ = strconv.Atoi(fd[:4])
				month, _ = strconv.Atoi(fd[4:])
			}
		case "Clase":
			var cls mensClase
			if err := dec.DecodeElement(&cls, &se); err != nil {
				return nil, err
			}
			if cls.ISIN == "" || year == 0 || month == 0 {
				continue
			}
			lastDay := daysInMonth(year, month)
			for day := 1; day <= lastDay; day++ {
				if cls.VLDiario[day] == 0 {
					continue
				}
				out = append(out, NAVObservation{
					ISIN:       cls.ISIN,
					Date:       fmt.Sprintf("%04d-%02d-%02d", year, month, day),
					NAV:        cls.VLDiario[day],
					Participes: cls.ParticipesDiario[day],
					Patrimonio: cls.PatrimonioDiario[day],
				})
			}
		}
	}
	return out, nil
}

func daysInMonth(year, month int) int {
	// time.Date normalizes out-of-range days; day 0 of (month+1) = last day of month.
	return time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.UTC).Day()
}
