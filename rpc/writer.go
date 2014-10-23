package rpc

/*
func pack(id int, v ...interface{}) Message {
	return Message{Data: v, Id: id}
}

func WriteOn(msg *Message, writer io.Writer) {
	//msg := &Message{Seed: seed, Data: data}

	switch msg.Call {
	case "compile":
		data := ethutil.NewValue(msg.Args)
		bcode, err := ethutil.Compile(data.Get(0).Str(), false)
		if err != nil {
			JSON.Send(writer, pack(msg.Id, err.Error()))
		}

		code := ethutil.Bytes2Hex(bcode)

		JSON.Send(writer, pack(msg.Id, code, nil))
	case "block":
		args := msg.Arguments()

		block := pipe.BlockByNumber(int32(args.Get(0).Uint()))

		JSON.Send(writer, pack(msg.Id, block))
	case "transact":
		if mp, ok := msg.Args[0].(map[string]interface{}); ok {
			object := mapToTxParams(mp)
			JSON.Send(
				writer,
				pack(msg.Id, args(pipe.Transact(object["from"], object["to"], object["value"], object["gas"], object["gasPrice"], object["data"]))),
			)

		}
	case "coinbase":
		JSON.Send(writer, pack(msg.Id, pipe.CoinBase(), msg.Seed))

	case "listening":
		JSON.Send(writer, pack(msg.Id, pipe.IsListening()))

	case "mining":
		JSON.Send(writer, pack(msg.Id, pipe.IsMining()))

	case "peerCoint":
		JSON.Send(writer, pack(msg.Id, pipe.PeerCount()))

	case "countAt":
		args := msg.Arguments()

		JSON.Send(writer, pack(msg.Id, pipe.TxCountAt(args.Get(0).Str())))

	case "codeAt":
		args := msg.Arguments()

		JSON.Send(writer, pack(msg.Id, len(pipe.CodeAt(args.Get(0).Str()))))

	case "stateAt":
		args := msg.Arguments()

		JSON.Send(writer, pack(msg.Id, pipe.StorageAt(args.Get(0).Str(), args.Get(1).Str())))

	case "balanceAt":
		args := msg.Arguments()

		JSON.Send(writer, pack(msg.Id, pipe.BalanceAt(args.Get(0).Str())))

	case "newFilter":
	case "newFilterString":
	case "messages":
		// TODO
	}
}
*/
