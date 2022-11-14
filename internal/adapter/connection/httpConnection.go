package connection

import (
	"danvieira97/go-webscraping/internal/domain"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func SearchDeputy() {

	deputys := searchAllDeputys()

	var wg sync.WaitGroup

	wg.Add(len(deputys))

	for i, dep := range deputys {

		time.Sleep(1 * time.Second)
		go func(id string, name string, policalParty string, state string) {
			defer wg.Done()

			fmt.Printf("Progresso: %d de %d\n", i, len(deputys))

			url := fmt.Sprintf("https://www.camara.leg.br/transparencia/gastos-parlamentares?legislatura=&ano=2021&mes=&por=deputado&deputado=%s&uf=&partido=", id)

			resp, err := http.Get(url)
			if err != nil {
				panic(err)
			}
			defer resp.Body.Close()

			doc, err := goquery.NewDocumentFromReader(resp.Body)
			if err != nil {
				log.Fatal(err)
			}

			var deputy domain.Deputy

			deputy.Name = name
			if err != nil {
				log.Fatal(err, "Deputado: ", name)
			}
			deputy.Name = name

			cabinetBudget, err := getCabinetBudget(*doc)
			if err != nil {
				log.Fatal(err, "Deputado: ", name)
			}
			deputy.CabinetBudget = cabinetBudget

			spentBudget, err := getSpentCabinetBudget(*doc)
			if err != nil {
				log.Fatal(err, "Deputado: ", name)
			}
			deputy.SpentCabinetBudget = spentBudget.SpentCabinetBudget
			deputy.SpentPercentage = spentBudget.SpentPercentage

			availableBudget, err := getAvailableCabinetBudget(*doc)
			if err != nil {
				log.Fatal(err, "Deputado: ", name)
			}
			deputy.AvailableCabinetBudget = availableBudget.AvailableCabinetBudget
			deputy.AvailablePercentage = availableBudget.AvailablePercentage
			deputy.PoliticalParty = policalParty
			deputy.State = state

			marshal, err := json.MarshalIndent(deputy, "", "")
			if err != nil {
				log.Fatalln(err.Error())
			}

			fmt.Printf("%s\n", marshal)

		}(dep.id, dep.name, dep.politicalParty, dep.state)
	}
	wg.Wait()
}

// func getName(doc goquery.Document) (string, error) {
// 	name := doc.Find("#main-content > section.gastos-form > div.gastos-form__resumo-resposta > div > p > span:nth-child(1)").Text()

// 	if len(name) == 0 {
// 		return "", errors.New("nome não encontrado")
// 	}

// 	return name, nil
// }

func getCabinetBudget(doc goquery.Document) (string, error) {
	cabinetBudget := doc.Find("#cota > div > div.l-cota__row > div:nth-child(1) > div > div.l-card.l-cota-resumo > div > div > section > p.gastos__resumo-texto.gastos__resumo-texto--destaque > span").Text()

	if len(cabinetBudget) == 0 {
		return "", errors.New("cota não encontrada")
	}

	return cabinetBudget, nil
}

func getSpentCabinetBudget(doc goquery.Document) (struct {
	SpentCabinetBudget string
	SpentPercentage    string
}, error) {
	spentCabinetBudget := doc.Find("#js-percentual-gasto > tbody > tr:nth-child(1) > td:nth-child(2)").Text()

	if len(spentCabinetBudget) == 0 {
		return struct {
			SpentCabinetBudget string
			SpentPercentage    string
		}{}, errors.New("verba gasta não encontrada")
	}

	spentPercentage := doc.Find("#js-percentual-gasto > tbody > tr:nth-child(1) > td:nth-child(3)").Text()

	if len(spentPercentage) == 0 {
		return struct {
			SpentCabinetBudget string
			SpentPercentage    string
		}{}, errors.New("porcentagem gasta não encontrada")
	}

	return struct {
		SpentCabinetBudget string
		SpentPercentage    string
	}{spentCabinetBudget, spentPercentage}, nil

}

func getAvailableCabinetBudget(doc goquery.Document) (struct {
	AvailableCabinetBudget string
	AvailablePercentage    string
}, error) {
	availableCabinetBudget := doc.Find("#js-percentual-gasto > tbody > tr:nth-child(2) > td:nth-child(2)").Text()

	if len(availableCabinetBudget) == 0 {
		return struct {
			AvailableCabinetBudget string
			AvailablePercentage    string
		}{}, errors.New("verba disponível não encontrada")
	}

	availablePercentage := doc.Find("#js-percentual-gasto > tbody > tr:nth-child(2) > td:nth-child(3)").Text()

	if len(availablePercentage) == 0 {
		return struct {
			AvailableCabinetBudget string
			AvailablePercentage    string
		}{}, errors.New("porcentagem disponível não encontrada")
	}

	return struct {
		AvailableCabinetBudget string
		AvailablePercentage    string
	}{availableCabinetBudget, availablePercentage}, nil
}

func searchAllDeputys() []struct {
	name           string
	politicalParty string
	state          string
	id             string
} {

	response, err := http.Get("https://www.camara.leg.br/transparencia/gastos-parlamentares")
	if err != nil {
		fmt.Printf("FALHA AO EXECUTAR REQUISICAO %d %s",
			response.StatusCode, response.Status)
		panic(err.Error())
	}
	defer response.Body.Close()

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var deputys []struct {
		name           string
		politicalParty string
		state          string
		id             string
	}

	doc.Find("#deputado").Each(func(i int, s *goquery.Selection) {
		s.Find("option").Each(func(i int, selection *goquery.Selection) {
			if len(selection.AttrOr("value", "")) != 0 {

				rgx := regexp.MustCompile("\\S+\\s\\S+")
				nome := rgx.FindString(selection.Text())

				rgx = regexp.MustCompile("\\(([^)]+)\\)")
				submatch := rgx.FindStringSubmatch(selection.Text())
				partidoEstado := submatch[1]
				partido := partidoEstado[0:2]
				estado := partidoEstado[len(partidoEstado)-2:]

				deputys = append(deputys, struct {
					name           string
					politicalParty string
					state          string
					id             string
				}{name: nome, politicalParty: partido, state: estado, id: selection.AttrOr("value", "")})
			}
		})
	})

	return deputys

}
