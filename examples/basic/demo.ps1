$brain = ".\agent.fbrain"

go run ./cmd/fmem init $brain
go run ./cmd/fmem remember $brain "Cliente Joao usa Firebird 2.5 e teve erro fiscal na NF-e apos atualizacao 3.2"
go run ./cmd/fmem remember $brain "Joao relatou novamente problema fiscal em nota eletronica depois da versao 3.2"
go run ./cmd/fmem recall $brain "problema fiscal NF-e"
go run ./cmd/fmem sync $brain
go run ./cmd/fmem context $brain "responder Joao sobre erro fiscal apos atualizacao"
