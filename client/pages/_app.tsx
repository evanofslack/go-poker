import React from 'react'
import { AppProps } from 'next/app'
import { SocketProvider } from '../providers/WebSocket'

import '../styles/index.css'

function MyApp({ Component, pageProps }: AppProps) {
  return (
    <SocketProvider>
      <Component {...pageProps} />
    </SocketProvider>
  )
}

export default MyApp