import React, { useCallback, useEffect, useState } from 'react';

export const isBrowser = typeof window !== "undefined";
const socket = isBrowser ? new WebSocket("ws://127.0.0.1:8080/ws") : null;


export default function Chat() {
  const [message, setMessage] = useState('')
  const [inputValue, setInputValue] = useState('')

  useEffect(() => {
    socket.onopen = () => {
      setMessage('Connected')
    };

    socket.onmessage = (e) => {
      setMessage("Get message from server: " + e.data)
    };

    return () => {
      socket.close()
    }
  }, [])

  const handleClick = useCallback((e) => {
    e.preventDefault()

    socket.send(JSON.stringify({
      state: inputValue
    }))
  }, [inputValue])

  const handleChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setInputValue(e.target.value)
  }, [])

  return (
    <div className="App">
      <input id="input" type="text" value={inputValue} onChange={handleChange} />
      <button onClick={handleClick}>Send</button>
      <pre>{message}</pre>
    </div>
  );
}