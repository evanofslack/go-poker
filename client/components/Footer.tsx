import { FiGithub } from "react-icons/fi";

export default function Footer() {
    return (
        <footer className=" text-sm font-light flex flex-row items-center justify-between w-screen py-3 px-8 pb-4 mt-10 md:mt-20 bg-white border-t-2 border-gray-100">
            <p> &copy; 2022 Samila </p>
            <a href="https://github.com/evanofslack/samila-ui">
                <FiGithub size="1.2rem" />
            </a>
        </footer>
    );
}
