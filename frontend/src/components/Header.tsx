import '../styles/Header.scss'

function Header(){
    return (
        <header className='header'>
            <h1>MOADA</h1>

            <div className='linkWrapper'>
                <a href="/"><img src="https://placehold.co/100x100" alt="" /></a>
                <a href="/account"><img src="https://placehold.co/100x100" alt="" /></a>
                <a href="/download"><img src="https://placehold.co/100x100" alt="" /></a>
                <a href=""><img src="https://placehold.co/100x100" alt="" /></a>
            </div>
        </header>
    )
}


export default Header