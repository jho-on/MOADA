import { Routes, Route } from 'react-router-dom'
import './styles/App.scss'
import Header from "./components/Header"
import Home from "./pages/Home"
import Account from './pages/Account'
import Download from './pages/Download'


function App() {
    return (
        <div>
            <Header/>
            <Routes>
                <Route path="/" element={<Home/>}/>
                <Route path="/account" element={<Account/>}/>
                <Route path="/download" element={<Download/>}/>
            </Routes>
        </div>
    )
}

export default App
