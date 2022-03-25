import { FiGithub } from "react-icons/fi";

export default function Footer() {
    return (
        <footer className=" mt-10 flex w-screen flex-row items-center justify-between border-t-2 border-gray-100 bg-white py-3 px-8 pb-4 text-sm font-light md:mt-20">
            <p> &copy; 2022 Samila </p>
            <a href="https://github.com/evanofslack/samila-ui">
                <FiGithub size="1.2rem" />
            </a>
        </footer>
    );
}
