import React, { useCallback, useContext, useEffect, useState } from 'react';
import { useSocket } from '../hooks/useSocket';
import { AppContext } from "../providers/AppStore"

export default function Join() {
  const socket = useSocket()
  const AppStore = useContext(AppContext)
  const [inputValue, setInputValue] = useState('')

  const handleClick = useCallback((e) => {
    e.preventDefault()

    AppStore.dispatch({type: "setUsername", payload:inputValue})

    socket?.send(JSON.stringify({
      action: "new-player",
      params: {
        username: inputValue,
      },
    }))
  }, [inputValue])

  const handleChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setInputValue(e.target.value)
  }, [])

  return (
    <div className="flex flex-col items-center justify-center h-screen">
      <h1 className="text-5xl font-semibold mb-16 text-white">PokerGo</h1>
        <div>
          <input className="bg-gray-600 text-white rounded-sm p-1 m-4" id="input" type="text" value={inputValue} placeholder="username" onChange={handleChange} />
          <button className="text-white bg-green-800 px-4 py-1 rounded-sm" onClick={handleClick}>Join</button>
        </div>
    </div>
  );
}