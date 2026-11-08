# Set Overlap Join

**Versão de desenvolvimento 0.1.0** — biblioteca e CLI original em Go desenvolvida pela [LOFRANO TECNOLOGIA](https://lofranotecnologia.com.br).  
**Repositório:** [martinlofranodeoliveira/set-overlap-join](https://github.com/martinlofranodeoliveira/set-overlap-join)

---

## Visão Geral

O `set-overlap-join` executa uma **junção exata por similaridade de Jaccard** em lotes de conjuntos pré-tokenizados. O algoritmo aplica filtros matematicamente comprovados de prefixo e cardinalidade para descartar pares não qualificáveis, seguido de verificação exata por números inteiros.

Ao contrário de abordagens probabilísticas (como MinHash, HyperLogLog ou SimHash), esta biblioteca assegura **100% de precisão e 100% de cobertura (recall)** sob um limiar racional $p/q$, sem aproximações de ponto flutuante e sem dependências externas em tempo de execução.

### Principais Garantias

- **Aritmética Inteira Exata**: Avalia a inequação $(q + p) \cdot |A \cap B| \ge p \cdot (|A| + |B|)$ evitando qualquer imprecisão de ponto flutuante.
- **Completude por Prefixos**: Descarta pares inviáveis pelo limite comprovado $|S| - \lceil p|S|/q \rceil + 1$, garantindo que nenhum par qualificável seja ignorado.
- **Tratamento Preciso de Conjuntos Vazios**: Define formalmente $J(\emptyset, \emptyset) = 1/1$ e $J(\emptyset, B) = 0$.
- **Orçamentos com Falha Segura (Fail-Closed)**: Contadores explícitos de visitas de prefixo e passos de junção permitem limitar a computação. Buscas interrompidas retornam `INCOMPLETE` com os pares válidos verificados até o momento, nunca informando um resultado incompleto como ausência de correspondências.
- **Saída Determinística**: Os pares retornados são ordenados canonicamente (`SetA < SetB`) e classificados de forma lexicográfica. Reordenar a entrada gera resultados idênticos byte a byte.

---

## Requisitos e Instalação

- **Go**: Versão 1.22 ou superior (apenas biblioteca padrão; sem dependências externas).
- **Sistema Operacional**: Linux, macOS ou Windows.

```bash
# Clonar o repositório
git clone https://github.com/martinlofranodeoliveira/set-overlap-join.git
cd set-overlap-join

# Executar testes unitários e de concorrência (race detector)
go test -v -race ./...

# Compilar o utilitário CLI
go build -o bin/set-overlap-join ./cmd/set-overlap-join
```

---

## Uso do Utilitário CLI

### 1. Demonstração Determinística Offline

Executa uma demonstração autocontida que valida os pares sob o limiar racional 2/3 em conjuntos sintéticos de rótulos de microsserviços:

```bash
# Saída formatada para leitura humana com validação de invariantes
./bin/set-overlap-join demo --assert

# Saída estruturada em JSON
./bin/set-overlap-join demo --assert --json
```

Exemplo de saída JSON:

```json
{
  "status": "COMPLETE_MATCHES",
  "pairs": [
    {
      "set_a": "empty:placeholder-a",
      "set_b": "empty:placeholder-b",
      "intersection": 0,
      "union": 0,
      "numerator": 1,
      "denominator": 1
    },
    {
      "set_a": "svc:auth-core",
      "set_b": "svc:auth-proxy",
      "intersection": 4,
      "union": 6,
      "numerator": 2,
      "denominator": 3
    }
  ],
  "stats": {
    "total_sets": 7,
    "empty_sets": 2,
    "non_empty_sets": 5,
    "prefix_visits": 9,
    "merge_steps": 9,
    "candidate_pairs": 2,
    "verified_pairs": 2
  }
}
```

### 2. Processando Arquivos de Entrada

```bash
./bin/set-overlap-join run --threshold 3/4 --input dados/conjuntos.json --json
```

Formato do arquivo JSON:

```json
{
  "sets": [
    {"id": "doc1", "tokens": [10, 20, 30, 40]},
    {"id": "doc2", "tokens": [20, 30, 40, 50]}
  ]
}
```

---

## Exemplo de Uso do SDK em Go

```go
package main

import (
	"fmt"
	"log"

	"github.com/martinlofranodeoliveira/set-overlap-join"
)

func main() {
	sets := []setjoin.Set{
		{ID: "conjunto_a", Tokens: []uint32{1, 2, 3, 4}},
		{ID: "conjunto_b", Tokens: []uint32{2, 3, 4, 5}},
	}

	threshold := setjoin.Threshold{P: 3, Q: 5} // Jaccard >= 3/5 (60%)
	budget := setjoin.Budget{}                 // Limites padrão

	result, err := setjoin.JoinSets(sets, threshold, budget)
	if err != nil {
		log.Fatalf("Erro na junção: %v", err)
	}

	fmt.Printf("Status: %s\n", result.Status)
	for _, pair := range result.Pairs {
		fmt.Printf("Par correspondente: %s <-> %s (Jaccard = %d/%d)\n",
			pair.SetA, pair.SetB, pair.Numerator, pair.Denominator)
	}
}
```

---

## Especificações e Documentação Técnica

- [Especificação de Contrato (docs/contract.md)](docs/contract.md): Definições matemáticas, prova inteira e limites de entrada.
- [Arquitetura e Design (docs/design.md)](docs/design.md): Índice invertido de prefixos, filtro de tamanho e disciplina de memória.

---

## Limites e Escopo

- Máximo de conjuntos admitidos: 1.024
- Máximo de tokens por conjunto: 1.024
- Máximo cumulativo de tokens no corpus: 65.536
- A biblioteca assume que os tokens já foram convertidos para inteiros `uint32`. Tokenização de texto em linguagem natural, embeddings e adaptadores de banco de dados não fazem parte do escopo deste projeto.

---

## Licença e Segurança

- **Licença**: MIT License. Copyright © 2026 Lofrano Tecnologia.
- **Relatório de Vulnerabilidades**: Utilize o [GitHub Private Vulnerability Reporting](https://github.com/martinlofranodeoliveira/set-overlap-join/security/advisories/new).
- **Página Oficial**: [lofranotecnologia.com.br](https://lofranotecnologia.com.br)
